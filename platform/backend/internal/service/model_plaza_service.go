package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// PlazaOfficialPricing 模型广场展示用的官方参考价（USD per token），与计费同源：
// LiteLLM → 内置兜底价卡 → 模型策略。字段为 nil 表示该项缺失（0 视为未配置）。
type PlazaOfficialPricing struct {
	InputPrice         *float64
	OutputPrice        *float64
	CacheWritePrice    *float64 // 5m 缓存写入（= LiteLLM cache_creation）
	CacheWrite1hPrice  *float64 // 1h 缓存写入，仅计费会区分 5m/1h 时给出
	CacheWrite12hPrice *float64 // OpenAI 12h cache-write price
	CacheReadPrice     *float64
	// Intervals 官方长上下文阶梯（多档时给出），不受分组开关影响。
	Intervals []PricingInterval
}

// PlazaModel 模型广场中单个模型条目：按实收口径合成的展示定价 + 官方参考价。
type PlazaModel struct {
	Name            string
	Platform        string
	Pricing         *ChannelModelPricing
	OfficialPricing *PlazaOfficialPricing
	// LongContextBasis 多档时的计价基准（整单 / 仅超出部分），单档为空。
	LongContextBasis ContextPricingBasis
	// TimePricing 计费会生效的分时倍率时段；无分时为 nil。
	TimePricing                   *TimePricingSchedule
	ServiceTiers                  []PlazaServiceTier
	CacheStoragePricePerTokenHour *float64
}

type PlazaServiceTier struct {
	Tier    string
	Pricing *ChannelModelPricing
}

type PlazaAdditionalCharges struct {
	WebSearchPerCall        float64 `json:"web_search_per_call"`
	SearchPer1k             float64 `json:"search_per_1k"`
	AudioRealtimePerMin     float64 `json:"audio_realtime_per_min"`
	AudioTTSPerMillionChars float64 `json:"audio_tts_per_million_chars"`
	AudioSTTPerHour         float64 `json:"audio_stt_per_hour"`
}

// PlazaGroup 模型广场中以分组为顶层的条目。
//
// 与 AvailableGroupRef 相比多了 Description 与 Models；Models 来自该分组关联渠道的
// 支持模型（普通分组按分组平台隔离，Composite 分组展开关联渠道已配置的
// 具体平台），与「可用渠道」页口径一致。
type PlazaGroup struct {
	ID                 int64
	Name               string
	Description        string
	I18n               GroupI18n
	Platform           string
	SubscriptionType   string
	RateMultiplier     float64
	PeakRateEnabled    bool
	PeakStart          string
	PeakEnd            string
	PeakRateMultiplier float64
	IsExclusive        bool
	// 图片按次实付倍率：ImageRateIndependent 为 true 时，图片计费模型的实付
	// = 档位价 × ImageRateMultiplier，不乘分组/用户专属倍率（与计费口径一致）。
	ImageRateIndependent bool
	ImageRateMultiplier  float64
	// 视频独立倍率与图片独立倍率分别配置，开启时覆盖分组/用户专属倍率。
	VideoRateIndependent bool
	VideoRateMultiplier  float64
	// LongContextPricingEnabled 分组是否按上下文长度应用阶梯价；关闭时模型展示的是最低档。
	LongContextPricingEnabled bool
	Models                    []PlazaModel
	AdditionalCharges         *PlazaAdditionalCharges
}

// ModelPlazaService 聚合模型广场数据。
//
// 模型枚举来自渠道配置；token 模型的展示单价与阶梯由 BillingService 的阶梯表
// 查询给出（与扣费走同一条解析链与计费函数），图片/按次模型沿用渠道/分组档位价。
type ModelPlazaService struct {
	channelRepo    ChannelRepository
	groupRepo      GroupRepository
	pricingService *PricingService
	billingService *BillingService
	resolver       *ModelPricingResolver
	modelLister    plazaModelLister
}

// Use the gateway's request-model listing, never upstream mapping targets or
// account metadata, when filling gaps in the operator's channel catalog.
type plazaModelLister interface {
	GetAvailableModels(context.Context, *int64, string) []string
}

func NewModelPlazaServiceWithGateway(channelRepo ChannelRepository, groupRepo GroupRepository, pricingService *PricingService, billingService *BillingService, resolver *ModelPricingResolver, gateway *GatewayService) *ModelPlazaService {
	s := NewModelPlazaService(channelRepo, groupRepo, pricingService, billingService, resolver)
	if gateway != nil {
		s.modelLister = gateway
	}
	return s
}

// NewModelPlazaService 创建模型广场服务。
func NewModelPlazaService(
	channelRepo ChannelRepository,
	groupRepo GroupRepository,
	pricingService *PricingService,
	billingService *BillingService,
	resolver *ModelPricingResolver,
) *ModelPlazaService {
	return &ModelPlazaService{
		channelRepo:    channelRepo,
		groupRepo:      groupRepo,
		pricingService: pricingService,
		billingService: billingService,
		resolver:       resolver,
	}
}

// ListGroups 返回模型广场数据：每个活跃分组附带其可用模型与定价。
//
// 模型枚举口径与 ListAvailable 一致（Active 渠道、SupportedModels ∪ 全局定价回落、
// 平台隔离），仅把顶层从渠道换成分组：
//   - 渠道按 lower(name) 排序后遍历，保证同名模型去重结果确定；
//   - 同分组同名模型「先见者胜」，仅当已存条目无定价而新条目有定价时升级替换；
//   - token 模型的单价与阶梯按实收口径合成（见 ResolveContextPricingSchedule），
//     图片计费模型的档位价按实收口径合成（见 plazaImageDisplayPricing）；
//   - 每个模型附带官方参考价（查不到为 nil）；
//   - 只返回 Models 非空的分组；分组按 RateMultiplier 升序（同倍率按名称），
//     组内模型按名称排序。
//
// 可见性过滤（专属分组）不在此层做，由 handler 按登录态裁剪。
func (s *ModelPlazaService) ListGroups(ctx context.Context) ([]PlazaGroup, error) {
	channels, err := s.channelRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active groups: %w", err)
	}

	sort.SliceStable(channels, func(i, j int) bool {
		return strings.ToLower(channels[i].Name) < strings.ToLower(channels[j].Name)
	})

	byGroup := make(map[int64]*PlazaGroup, len(groups))
	groupEnt := make(map[int64]*Group, len(groups))
	order := make([]int64, 0, len(groups))
	for i := range groups {
		g := &groups[i]
		byGroup[g.ID] = &PlazaGroup{
			ID:                        g.ID,
			Name:                      g.Name,
			Description:               g.Description,
			I18n:                      g.I18n,
			Platform:                  g.Platform,
			SubscriptionType:          g.SubscriptionType,
			RateMultiplier:            g.RateMultiplier,
			PeakRateEnabled:           g.PeakRateEnabled,
			PeakStart:                 g.PeakStart,
			PeakEnd:                   g.PeakEnd,
			PeakRateMultiplier:        g.PeakRateMultiplier,
			IsExclusive:               g.IsExclusive,
			ImageRateIndependent:      g.ImageRateIndependent,
			ImageRateMultiplier:       g.ImageRateMultiplier,
			VideoRateIndependent:      g.VideoRateIndependent,
			VideoRateMultiplier:       g.VideoRateMultiplier,
			LongContextPricingEnabled: g.LongContextPricingEnabled,
		}
		groupEnt[g.ID] = g
		order = append(order, g.ID)
	}

	type modelKey struct {
		platform string
		name     string
	}
	// modelIdx[groupID][platform+modelName] = index into byGroup[groupID].Models
	modelIdx := make(map[int64]map[modelKey]int, len(groups))
	for i := range channels {
		ch := &channels[i]
		if ch.Status != StatusActive {
			continue
		}
		ch.normalizeBillingModelSource()
		supported := ch.SupportedModels()
		fillGlobalPricingFallback(s.pricingService, supported)

		for _, gid := range ch.GroupIDs {
			pg, ok := byGroup[gid]
			if !ok {
				continue
			}
			idx := modelIdx[gid]
			if idx == nil {
				idx = make(map[modelKey]int, len(supported))
				modelIdx[gid] = idx
			}
			for j := range supported {
				m := supported[j]
				if pg.Platform == PlatformComposite {
					if !isConcreteRequestPlatform(m.Platform) {
						continue
					}
				} else if m.Platform != pg.Platform {
					continue
				}
				key := modelKey{platform: m.Platform, name: m.Name}
				if at, seen := idx[key]; seen {
					// 先见者胜；仅当已存条目无定价而新条目有定价时升级。
					if pg.Models[at].Pricing == nil && m.Pricing != nil {
						pg.Models[at].Pricing = m.Pricing
					}
					continue
				}
				idx[key] = len(pg.Models)
				pg.Models = append(pg.Models, PlazaModel{
					Name:     m.Name,
					Platform: m.Platform,
					Pricing:  m.Pricing,
				})
			}
		}
	}

	officialMemo := make(map[string]*PlazaOfficialPricing)
	out := make([]PlazaGroup, 0, len(order))
	for _, gid := range order {
		pg := byGroup[gid]
		g := groupEnt[gid]
		for i := range g.ModelPricing {
			card := g.ModelPricing[i].Clone()
			platform := g.Platform
			if platform == PlatformComposite {
				platform = card.Platform
			}
			if !isConcreteRequestPlatform(platform) {
				continue
			}
			for _, name := range card.Models {
				name = strings.TrimSpace(name)
				if name == "" || strings.Contains(name, "*") {
					continue
				}
				found := false
				for _, existing := range pg.Models {
					if existing.Platform == platform && strings.EqualFold(existing.Name, name) {
						found = true
						break
					}
				}
				if !found {
					pg.Models = append(pg.Models, PlazaModel{Name: name, Platform: platform, Pricing: &card})
				}
			}
		}
		if s.billingService != nil {
			pg.AdditionalCharges = &PlazaAdditionalCharges{
				WebSearchPerCall:        s.billingService.CalculateWebSearchCost(1, g.WebSearchPricePerCall, 1).TotalCost,
				SearchPer1k:             s.billingService.CalculateSearchCost(1000, g.SearchPricePer1k, 1).TotalCost,
				AudioRealtimePerMin:     s.billingService.CalculateAudioCost("realtime", 1, &audioPriceConfig{RealtimePerMin: g.AudioRealtimePricePerMin}, 1).TotalCost,
				AudioTTSPerMillionChars: s.billingService.CalculateAudioCost("tts", 1, &audioPriceConfig{TTSPerMChars: g.AudioTTSPricePerMillionChars}, 1).TotalCost,
				AudioSTTPerHour:         s.billingService.CalculateAudioCost("stt", 1, &audioPriceConfig{STTPerHour: g.AudioSTTPricePerHour}, 1).TotalCost,
			}
		}
		s.fillServingModels(ctx, pg, g)
		allowed := pg.Models[:0]
		for _, m := range pg.Models {
			if g.ModelAllowlist.Allows(m.Name) && !s.plazaModelRestricted(ctx, g, m.Platform, m.Name) {
				allowed = append(allowed, m)
			}
		}
		pg.Models = allowed
		if len(pg.Models) == 0 {
			continue
		}
		sort.SliceStable(pg.Models, func(i, j int) bool {
			if pg.Models[i].Name != pg.Models[j].Name {
				return pg.Models[i].Name < pg.Models[j].Name
			}
			return pg.Models[i].Platform < pg.Models[j].Platform
		})
		for j := range pg.Models {
			publicName := pg.Models[j].Name
			pg.Models[j].Name = s.plazaBillingModel(ctx, g, pg.Models[j].Platform, publicName)
			s.fillDisplayPricing(ctx, &pg.Models[j], g)
			s.fillServiceTierPricing(ctx, &pg.Models[j], g)
			if s.billingService != nil {
				if price := s.billingService.CacheStoragePricePerTokenHour(pg.Models[j].Name); price > 0 {
					pg.Models[j].CacheStoragePricePerTokenHour = &price
				}
			}
			pg.Models[j].OfficialPricing = s.lookupOfficialPricing(ctx, pg.Models[j].Name, officialMemo)
			pg.Models[j].Name = publicName
		}
		out = append(out, *pg)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RateMultiplier != out[j].RateMultiplier {
			return out[i].RateMultiplier < out[j].RateMultiplier
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func (s *ModelPlazaService) fillServiceTierPricing(ctx context.Context, m *PlazaModel, g *Group) {
	if s.billingService == nil || s.resolver == nil || m.Pricing == nil || (m.Pricing.BillingMode != "" && m.Pricing.BillingMode != BillingModeToken) {
		return
	}
	pricingCtx := WithResolvedTargetPlatform(ctx, m.Platform)
	resolved := s.resolver.Resolve(pricingCtx, PricingInput{Model: m.Name, Group: g, GroupID: &g.ID})
	if resolved == nil || resolved.BasePricing == nil {
		return
	}
	p := resolved.BasePricing
	var tiers []string
	if p.FastMultiplier != nil || p.InputPricePerTokenPriority > 0 || p.OutputPricePerTokenPriority > 0 {
		tiers = append(tiers, "priority")
	}
	if m.Platform == PlatformOpenAI && strings.HasPrefix(strings.ToLower(m.Name), "gpt-") {
		tiers = append(tiers, "flex")
	}
	if p.UltrafastMultiplier > 0 {
		tiers = append(tiers, "ultrafast")
	}
	for _, tier := range tiers {
		billingTier := tier
		if g.FreeOpenAIFast && m.Platform == PlatformOpenAI && tier == "priority" {
			billingTier = "default"
		}
		sched, err := s.billingService.ResolveContextPricingSchedule(ctx, s.resolver, ContextPricingScheduleInput{Model: m.Name, Group: g, Platform: m.Platform, ServiceTier: billingTier})
		if err == nil && sched != nil && len(sched.Tiers) > 0 {
			m.ServiceTiers = append(m.ServiceTiers, PlazaServiceTier{Tier: tier, Pricing: plazaPricingFromSchedule(m.Pricing, sched)})
		}
	}
}

func (s *ModelPlazaService) plazaModelRestricted(ctx context.Context, g *Group, platform, model string) bool {
	if s.resolver == nil || s.resolver.channelService == nil {
		return false
	}
	ctx = WithResolvedTargetPlatform(ctx, platform)
	mapping := s.resolver.channelService.ResolveChannelMapping(ctx, g.ID, model)
	billingModel := billingModelForRestriction(mapping.BillingModelSource, model, mapping.MappedModel)
	return billingModel != "" && s.resolver.channelService.IsModelRestricted(ctx, g.ID, billingModel)
}

func (s *ModelPlazaService) plazaBillingModel(ctx context.Context, g *Group, platform, model string) string {
	if s.resolver == nil || s.resolver.channelService == nil {
		return model
	}
	mapping := s.resolver.channelService.ResolveChannelMapping(WithResolvedTargetPlatform(ctx, platform), g.ID, model)
	if mapping.BillingModelSource != BillingModelSourceRequested && mapping.MappedModel != "" {
		return mapping.MappedModel
	}
	return model
}

// Fill only concrete, accepted request names with an effective token price.
// Composite groups retain their explicitly configured per-platform catalog;
// wildcard mappings and unpriced/custom media names are not published as SKUs.
func (s *ModelPlazaService) fillServingModels(ctx context.Context, pg *PlazaGroup, g *Group) {
	if s.modelLister == nil || s.billingService == nil || s.resolver == nil || g.Platform == PlatformComposite {
		return
	}
	seen := make(map[string]bool, len(pg.Models))
	for _, m := range pg.Models {
		seen[strings.ToLower(m.Platform+"/"+m.Name)] = true
	}
	for _, name := range s.modelLister.GetAvailableModels(ctx, &g.ID, g.Platform) {
		name = strings.TrimSpace(name)
		key := strings.ToLower(g.Platform + "/" + name)
		if name == "" || strings.Contains(name, "*") || seen[key] || !g.ModelAllowlist.Allows(name) || s.plazaModelRestricted(ctx, g, g.Platform, name) {
			continue
		}
		if !g.AllowImageGeneration && isOpenAIImageGenerationModel(name) {
			continue
		}
		billingModel := s.plazaBillingModel(ctx, g, g.Platform, name)
		sched, err := s.billingService.ResolveContextPricingSchedule(ctx, s.resolver, ContextPricingScheduleInput{Model: billingModel, Group: g, Platform: g.Platform})
		if err != nil || sched == nil || len(sched.Tiers) == 0 {
			continue
		}
		seen[key] = true
		pg.Models = append(pg.Models, PlazaModel{Name: name, Platform: g.Platform, Pricing: plazaPricingFromSchedule(nil, sched)})
	}
}

// fillDisplayPricing 把模型的展示定价换成实收口径：
// token 模型取计费阶梯表（单价与档位均由真实计费函数得出），
// 图片/按次模型（或阶梯表不可用时）沿用渠道定价与分组图片档位价。
func (s *ModelPlazaService) fillDisplayPricing(ctx context.Context, m *PlazaModel, g *Group) {
	if groupPricing := matchGroupModelPricing(g, m.Name); groupPricing != nil {
		m.Pricing = groupPricing
		// 分组模型级固定价就是实收价，不能再被分组图片兜底档位覆盖。
		// 否则模型广场会显示 image_price_1k/2k/4k，而实际按 model_pricing 扣费。
		if groupPricing.BillingMode == BillingModeImage || groupPricing.BillingMode == BillingModePerRequest {
			return
		}
	}
	if s.billingService != nil && s.resolver != nil {
		sched, err := s.billingService.ResolveContextPricingSchedule(ctx, s.resolver, ContextPricingScheduleInput{
			Model:    m.Name,
			Group:    g,
			Platform: m.Platform,
		})
		if err == nil && sched != nil && len(sched.Tiers) > 0 {
			m.Pricing = plazaPricingFromSchedule(m.Pricing, sched)
			if len(sched.Tiers) > 1 {
				m.LongContextBasis = sched.Basis
			}
			m.TimePricing = sched.TimePricing
			return
		}
	}
	m.Pricing = plazaImageDisplayPricing(m.Pricing, g)
}

// plazaPricingFromSchedule 把阶梯表压成展示用的 ChannelModelPricing：
// 平价取首档单价，多档时 Intervals 逐档给出绝对单价；图片/按次字段沿用原始定价。
func plazaPricingFromSchedule(raw *ChannelModelPricing, sched *ContextPricingSchedule) *ChannelModelPricing {
	out := &ChannelModelPricing{BillingMode: BillingModeToken}
	if raw != nil {
		out.ImageInputPrice = raw.ImageInputPrice
		out.ImageOutputPrice = raw.ImageOutputPrice
		out.PerRequestPrice = raw.PerRequestPrice
		out.ReasoningEffortMultipliers = reasoningEffortMultipliersFromPricing(raw)
	}
	first := sched.Tiers[0]
	out.InputPrice = first.Input
	out.OutputPrice = first.Output
	out.CacheWritePrice = first.CacheWrite
	out.CacheWrite1hPrice = first.CacheWrite1h
	out.CacheWrite12hPrice = first.CacheWrite12h
	out.CacheReadPrice = first.CacheRead
	if len(sched.Tiers) > 1 {
		out.Intervals = plazaIntervalsFromTiers(sched.Tiers)
	}
	return out
}

func plazaIntervalsFromTiers(tiers []ContextPricingTier) []PricingInterval {
	intervals := make([]PricingInterval, 0, len(tiers))
	for i, t := range tiers {
		intervals = append(intervals, PricingInterval{
			MinTokens:          t.MinTokens,
			MaxTokens:          t.MaxTokens,
			TierLabel:          t.Label,
			InputPrice:         t.Input,
			OutputPrice:        t.Output,
			CacheWritePrice:    t.CacheWrite,
			CacheWrite1hPrice:  t.CacheWrite1h,
			CacheWrite12hPrice: t.CacheWrite12h,
			CacheReadPrice:     t.CacheRead,
			SortOrder:          i,
		})
	}
	return intervals
}

// plazaImageDisplayPricing 为图片计费模型合成展示定价，使档位价与实收口径一致：
// 每档（1K/2K/4K）单价 = 分组图片价 > 渠道同档位价 > 渠道默认按次价，无价的档不展示。
// 分组未配任何图片价、或定价非图片模式时原样返回。返回克隆，不修改入参
// （渠道定价指针指向缓存共享数据）。
func plazaImageDisplayPricing(p *ChannelModelPricing, g *Group) *ChannelModelPricing {
	if p == nil || g == nil || p.BillingMode != BillingModeImage {
		return p
	}
	if g.ImagePrice1K == nil && g.ImagePrice2K == nil && g.ImagePrice4K == nil {
		return p
	}
	channelTierPrice := func(label string) *float64 {
		for i := range p.Intervals {
			if p.Intervals[i].TierLabel == label && p.Intervals[i].PerRequestPrice != nil {
				return p.Intervals[i].PerRequestPrice
			}
		}
		return p.PerRequestPrice
	}
	tiers := []struct {
		label      string
		groupPrice *float64
	}{
		{"1K", g.ImagePrice1K},
		{"2K", g.ImagePrice2K},
		{"4K", g.ImagePrice4K},
	}
	clone := *p
	clone.Intervals = make([]PricingInterval, 0, len(tiers))
	for i, t := range tiers {
		price := t.groupPrice
		if price == nil {
			price = channelTierPrice(t.label)
		}
		if price == nil {
			continue
		}
		v := *price
		clone.Intervals = append(clone.Intervals, PricingInterval{
			TierLabel:       t.label,
			PerRequestPrice: &v,
			SortOrder:       i,
		})
	}
	return &clone
}

// lookupOfficialPricing 查询模型的官方参考价（与计费同源：LiteLLM → 内置兜底 → 模型策略），
// 带 memo 避免同名模型重复解析。官方阶梯按无分组、无渠道的口径查阶梯表。
// billingService 为 nil（测试场景）或查不到时返回 nil。
func (s *ModelPlazaService) lookupOfficialPricing(ctx context.Context, modelName string, memo map[string]*PlazaOfficialPricing) *PlazaOfficialPricing {
	if s.billingService == nil {
		return nil
	}
	if cached, ok := memo[modelName]; ok {
		return cached
	}
	var result *PlazaOfficialPricing
	if mp, err := s.billingService.GetModelPricing(modelName); err == nil && mp != nil {
		result = &PlazaOfficialPricing{
			InputPrice:      nonZeroPtr(mp.InputPricePerToken),
			OutputPrice:     nonZeroPtr(mp.OutputPricePerToken),
			CacheWritePrice: nonZeroPtr(mp.CacheCreationPricePerToken),
			CacheReadPrice:  nonZeroPtr(mp.CacheReadPricePerToken),
		}
		// 计费只在支持 5m/1h 分档时使用 1h 价，其余情况 1h 价对用户无意义。
		if mp.SupportsCacheBreakdown {
			result.CacheWrite1hPrice = nonZeroPtr(mp.CacheCreation1hPrice)
		}
		result.CacheWrite12hPrice = mp.CacheCreation12hPricePerToken
		if s.resolver != nil {
			sched, schedErr := s.billingService.ResolveContextPricingSchedule(ctx, s.resolver, ContextPricingScheduleInput{Model: modelName})
			if schedErr == nil && sched != nil && len(sched.Tiers) > 1 {
				result.Intervals = plazaIntervalsFromTiers(sched.Tiers)
			}
		}
		if result.InputPrice == nil && result.OutputPrice == nil && result.CacheWritePrice == nil &&
			result.CacheWrite1hPrice == nil && result.CacheWrite12hPrice == nil && result.CacheReadPrice == nil && len(result.Intervals) == 0 {
			result = nil
		}
	}
	memo[modelName] = result
	return result
}
