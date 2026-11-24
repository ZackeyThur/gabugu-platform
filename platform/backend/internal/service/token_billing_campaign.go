package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

const (
	SettingKeyTokenBillingCampaign         = "token_billing_campaign"
	defaultTokenBillingCampaignCacheTTL    = 15 * time.Second
	tokenBillingCampaignSingleflightKey    = "token_billing_campaign"
)

type cachedTokenBillingCampaign struct {
	campaign  *TokenBillingCampaign
	fetchedAt time.Time
}

// TokenBillingCampaign is an independent customer token discount. It must never
// change group/user rates, upstream prices, account statistics, or admission quotas.
type TokenBillingCampaign struct {
	SchemaVersion int                         `json:"schema_version"`
	ID            string                      `json:"id"`
	Revision      int                         `json:"revision"`
	Enabled       bool                        `json:"enabled"`
	StartsAt      time.Time                   `json:"starts_at"`
	Phases        []TokenBillingCampaignPhase `json:"phases"`
}
type TokenBillingCampaignPhase struct {
	ID                 string             `json:"id"`
	StartOffsetSeconds int64              `json:"start_offset_seconds"`
	EndOffsetSeconds   int64              `json:"end_offset_seconds"`
	GroupMultipliers   map[string]float64 `json:"group_multipliers"`
}

// TokenCampaignUsage is persisted with each billed row, so later config edits
// cannot rewrite the explanation of a historical balance change.
type TokenCampaignUsage struct {
	ID                 string    `json:"id"`
	Revision           int       `json:"revision"`
	Phase              string    `json:"phase"`
	Discount           float64   `json:"discount"`
	PricingAt          time.Time `json:"pricing_at"`
	OriginalActualCost float64   `json:"original_actual_cost"`
	DiscountAmount     float64   `json:"discount_amount"`
}
type PublicBillingCampaignGroup struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	DayDiscount  float64 `json:"day_discount"`
	WeekDiscount float64 `json:"week_discount"`
}
type PublicBillingCampaign struct {
	ID           string                       `json:"id"`
	Title        string                       `json:"title"`
	PricingBasis string                       `json:"pricing_basis"`
	StartsAt     time.Time                    `json:"starts_at"`
	DayEndsAt    time.Time                    `json:"day_ends_at"`
	EndsAt       time.Time                    `json:"ends_at"`
	Groups       []PublicBillingCampaignGroup `json:"groups"`
}
type PublicBillingCampaignResponse struct {
	Campaign  *PublicBillingCampaign `json:"campaign"`
	ServerNow time.Time              `json:"server_now"`
	Status    string                 `json:"status"`
}

var tokenCampaignGroupNames = map[int64]string{4: "GPT", 5: "Claude", 23: "DeepSeek", 24: "Gemini"}

func parseTokenBillingCampaign(raw string) (*TokenBillingCampaign, error) {
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	var c TokenBillingCampaign
	if err := decoder.Decode(&c); err != nil {
		return nil, errors.New("invalid token billing campaign JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("invalid token billing campaign trailing data")
	}
	if c.SchemaVersion != 1 || c.ID != "blg-20261115" || c.Revision < 1 || c.StartsAt.IsZero() || len(c.Phases) != 2 {
		return nil, errors.New("invalid token billing campaign identity")
	}
	for i, p := range c.Phases {
		expectedID := "championship-day"
		start, end := int64(0), int64(86400)
		if i == 1 {
			expectedID = "celebration-week"
			start, end = 86400, 691200
		}
		if p.ID != expectedID || p.StartOffsetSeconds != start || p.EndOffsetSeconds != end || len(p.GroupMultipliers) != 4 {
			return nil, errors.New("invalid token billing campaign phases")
		}
		for gid := range tokenCampaignGroupNames {
			factor, ok := p.GroupMultipliers[strconv.FormatInt(gid, 10)]
			expected := .9
			if i == 0 {
				expected = .5
				if gid == 4 {
					expected = .2
				}
			}
			if !ok || math.IsNaN(factor) || math.IsInf(factor, 0) || factor != expected {
				return nil, errors.New("invalid token billing campaign group discounts")
			}
		}
	}
	return &c, nil
}
func (s *SettingService) tokenBillingCampaignTTLValue() time.Duration {
	if s != nil && s.tokenBillingCampaignTTL > 0 {
		return s.tokenBillingCampaignTTL
	}
	return defaultTokenBillingCampaignCacheTTL
}

func (s *SettingService) tokenBillingCampaignNowTime() time.Time {
	if s != nil && s.tokenBillingCampaignNow != nil {
		return s.tokenBillingCampaignNow()
	}
	return time.Now()
}

// InvalidateTokenBillingCampaignCache clears the in-process token billing campaign cache.
func (s *SettingService) InvalidateTokenBillingCampaignCache() {
	if s == nil {
		return
	}
	s.tokenBillingCampaignSF.Forget(tokenBillingCampaignSingleflightKey)
	s.tokenBillingCampaignCache.Store((*cachedTokenBillingCampaign)(nil))
}

func (s *SettingService) tokenBillingCampaign(ctx context.Context) (*TokenBillingCampaign, error) {
	if s == nil || s.settingRepo == nil {
		return nil, nil
	}
	now := s.tokenBillingCampaignNowTime()
	ttl := s.tokenBillingCampaignTTLValue()
	if cached, ok := s.tokenBillingCampaignCache.Load().(*cachedTokenBillingCampaign); ok && cached != nil {
		if now.Sub(cached.fetchedAt) < ttl {
			return cached.campaign, nil
		}
	}

	val, err, _ := s.tokenBillingCampaignSF.Do(tokenBillingCampaignSingleflightKey, func() (any, error) {
		now := s.tokenBillingCampaignNowTime()
		if cached, ok := s.tokenBillingCampaignCache.Load().(*cachedTokenBillingCampaign); ok && cached != nil {
			if now.Sub(cached.fetchedAt) < ttl {
				return cached.campaign, nil
			}
		}
		baseCtx := context.Background()
		if ctx != nil {
			baseCtx = context.WithoutCancel(ctx)
		}
		dbCtx, cancel := context.WithTimeout(baseCtx, gatewayForwardingDBTimeout)
		defer cancel()
		raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyTokenBillingCampaign)
		if errors.Is(err, ErrSettingNotFound) || (err == nil && strings.TrimSpace(raw) == "") {
			s.tokenBillingCampaignCache.Store(&cachedTokenBillingCampaign{
				campaign:  nil,
				fetchedAt: s.tokenBillingCampaignNowTime(),
			})
			return (*TokenBillingCampaign)(nil), nil
		}
		if err != nil {
			return nil, errors.New("token billing campaign configuration unavailable")
		}
		parsed, parseErr := parseTokenBillingCampaign(raw)
		if parseErr != nil {
			return nil, parseErr
		}
		s.tokenBillingCampaignCache.Store(&cachedTokenBillingCampaign{
			campaign:  parsed,
			fetchedAt: s.tokenBillingCampaignNowTime(),
		})
		return parsed, nil
	})
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	return val.(*TokenBillingCampaign), nil
}
func (c *TokenBillingCampaign) phaseAt(at time.Time) *TokenBillingCampaignPhase {
	if c == nil || !c.Enabled || at.Before(c.StartsAt) {
		return nil
	}
	elapsed := at.Sub(c.StartsAt)
	for i := range c.Phases {
		p := &c.Phases[i]
		if elapsed >= time.Duration(p.StartOffsetSeconds)*time.Second && elapsed < time.Duration(p.EndOffsetSeconds)*time.Second {
			return p
		}
	}
	return nil
}
func (s *SettingService) GetPublicBillingCampaign(ctx context.Context, at time.Time) PublicBillingCampaignResponse {
	out := PublicBillingCampaignResponse{ServerNow: at, Status: "disabled"}
	c, err := s.tokenBillingCampaign(ctx)
	if err != nil {
		out.Status = "invalid"
		return out
	}
	if c == nil || !c.Enabled {
		return out
	}
	public := &PublicBillingCampaign{ID: c.ID, Title: "BLG夺冠，GABUGU兑现。", PricingBasis: "site_standard", StartsAt: c.StartsAt, DayEndsAt: c.StartsAt.Add(24 * time.Hour), EndsAt: c.StartsAt.Add(192 * time.Hour)}
	for _, gid := range []int64{4, 5, 23, 24} {
		key := strconv.FormatInt(gid, 10)
		public.Groups = append(public.Groups, PublicBillingCampaignGroup{ID: gid, Name: tokenCampaignGroupNames[gid], DayDiscount: c.Phases[0].GroupMultipliers[key], WeekDiscount: c.Phases[1].GroupMultipliers[key]})
	}
	out.Campaign = public
	out.Status = "scheduled"
	if p := c.phaseAt(at); p != nil {
		out.Status = p.ID
	} else if !at.Before(public.EndsAt) {
		out.Status = "ended"
	}
	return out
}
func (r PublicBillingCampaignResponse) TokenDiscount(groupID int64, subscriptionType string) float64 {
	if subscriptionType == "subscription" || r.Campaign == nil {
		return 1
	}
	for _, g := range r.Campaign.Groups {
		if g.ID == groupID {
			switch r.Status {
			case "championship-day":
				return g.DayDiscount
			case "celebration-week":
				return g.WeekDiscount
			}
		}
	}
	return 1
}

type tokenCampaignPricingCtxKey struct{}
type tokenCampaignPricing struct {
	groupID int64
	usage   TokenCampaignUsage
	invalid bool
}

// freezeTokenCampaign always stamps a value, including absent/disabled config.
// Settlement consumes only this immutable request/turn value and never rereads settings.
func (s *SettingService) freezeTokenCampaign(ctx context.Context, groupID *int64, at time.Time) context.Context {
	frozen := tokenCampaignPricing{}
	if groupID == nil {
		return context.WithValue(ctx, tokenCampaignPricingCtxKey{}, frozen)
	}
	frozen.groupID = *groupID
	if _, ok := tokenCampaignGroupNames[*groupID]; !ok {
		return context.WithValue(ctx, tokenCampaignPricingCtxKey{}, frozen)
	}
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && group != nil && group.ID == *groupID && group.IsSubscriptionType() {
		return context.WithValue(ctx, tokenCampaignPricingCtxKey{}, frozen)
	}
	c, err := s.tokenBillingCampaign(ctx)
	if err != nil {
		frozen.invalid = true
	} else if p := c.phaseAt(at); p != nil {
		frozen.usage = TokenCampaignUsage{ID: c.ID, Revision: c.Revision, Phase: p.ID, Discount: p.GroupMultipliers[strconv.FormatInt(*groupID, 10)], PricingAt: at}
	}
	return context.WithValue(ctx, tokenCampaignPricingCtxKey{}, frozen)
}
func tokenCampaignPricingError(ctx context.Context) error {
	if frozen, ok := ctx.Value(tokenCampaignPricingCtxKey{}).(tokenCampaignPricing); ok && frozen.invalid {
		return fmt.Errorf("%w: token campaign configuration unavailable", ErrDispatchPricingUnavailable)
	}
	return nil
}

// TokenCampaignPricingError is used by protocol handlers that must reject
// before upstream side effects (including explicit-cache create/patch).
func TokenCampaignPricingError(ctx context.Context) error { return tokenCampaignPricingError(ctx) }
func CopyTokenCampaignPricingContext(parent, base context.Context) context.Context {
	if parent != nil {
		if frozen, ok := parent.Value(tokenCampaignPricingCtxKey{}).(tokenCampaignPricing); ok {
			base = context.WithValue(base, tokenCampaignPricingCtxKey{}, frozen)
		}
	}
	return base
}
func (s *OpenAIGatewayService) WithTokenCampaignPricingAt(ctx context.Context, groupID *int64, at time.Time) context.Context {
	return s.settingService.freezeTokenCampaign(ctx, groupID, at)
}
func (s *GatewayService) WithTokenCampaignPricingAt(ctx context.Context, groupID *int64, at time.Time) context.Context {
	return s.settingService.freezeTokenCampaign(ctx, groupID, at)
}
func (s *GatewayService) WithGatewayTokenRequestPricing(ctx context.Context) (context.Context, time.Time) {
	ctx, at := WithGatewayTokenRequestPricing(ctx)
	var gid *int64
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && group != nil {
		gid = &group.ID
	}
	return s.settingService.freezeTokenCampaign(ctx, gid, at), at
}

// captureTokenCampaignEligibleCost runs on the token bucket before additive
// search/storage charges. Image token buckets retain their original price.
func captureTokenCampaignEligibleCost(cost *CostBreakdown) *CostBreakdown {
	if cost == nil || cost.TotalCost <= 0 || (cost.BillingMode != "" && cost.BillingMode != string(BillingModeToken)) {
		return cost
	}
	eligible := cost.InputCost + cost.OutputCost + cost.CacheCreationCost + cost.CacheReadCost
	ratio := math.Min(1, math.Max(0, eligible/cost.TotalCost))
	cost.tokenCampaignEligibleActualCost = cost.ActualCost * ratio
	return cost
}
func applyTokenCampaign(ctx context.Context, key *APIKey, subscription bool, cost *CostBreakdown) (*TokenCampaignUsage, error) {
	if subscription || key == nil || key.GroupID == nil || (key.Group != nil && key.Group.IsSubscriptionType()) {
		return nil, nil
	}
	frozen, ok := ctx.Value(tokenCampaignPricingCtxKey{}).(tokenCampaignPricing)
	if !ok || frozen.groupID != *key.GroupID {
		return nil, nil
	}
	if frozen.invalid {
		return nil, tokenCampaignPricingError(ctx)
	}
	if frozen.usage.ID == "" || cost == nil || cost.tokenCampaignEligibleActualCost <= 0 {
		return nil, nil
	}
	if math.IsNaN(cost.ActualCost) || math.IsInf(cost.ActualCost, 0) || cost.ActualCost < 0 || math.IsNaN(cost.tokenCampaignEligibleActualCost) || math.IsInf(cost.tokenCampaignEligibleActualCost, 0) {
		return nil, errors.New("invalid token campaign billing amount")
	}
	snapshot := frozen.usage
	snapshot.OriginalActualCost = cost.ActualCost
	eligible := math.Min(cost.ActualCost, cost.tokenCampaignEligibleActualCost)
	snapshot.DiscountAmount = eligible * (1 - snapshot.Discount)
	cost.ActualCost -= snapshot.DiscountAmount
	return &snapshot, nil
}
