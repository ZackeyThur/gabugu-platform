package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

// PricingPreflightKind identifies the billing path that will consume the
// selected account. It keeps media checks separate from token checks.
type PricingPreflightKind int

const (
	PricingPreflightToken PricingPreflightKind = iota
	PricingPreflightImage
	PricingPreflightVideo
	PricingPreflightAudio
	PricingPreflightPerRequest
)

var ErrDispatchPricingUnavailable = errors.New("dispatch pricing unavailable")

// ContextValueGetter is the small part of gin.Context needed by the dispatch
// guard. Keeping it structural avoids coupling the service package to handler
// middleware while still reading the authenticated API key.
type ContextValueGetter interface {
	Get(string) (any, bool)
}

// PricingPreflightInput is evaluated after account selection and before any
// upstream request is sent. Account mappings are included because the model
// billed after dispatch may differ from the requested model.
type PricingPreflightInput struct {
	Resolver   *ModelPricingResolver
	Billing    *BillingService
	Group      *Group
	GroupID    *int64
	Account    *Account
	Model      string
	Kind       PricingPreflightKind
	Candidates []string
}

// CheckDispatchPricing verifies that at least one model candidate has an
// explicit, usable price for the selected billing path. An explicit zero price
// in a configured tier remains valid; nil pricing remains unavailable.
func CheckDispatchPricing(ctx context.Context, in PricingPreflightInput) error {
	if err := tokenCampaignPricingError(ctx); err != nil {
		return err
	}
	if strings.TrimSpace(in.Model) == "" {
		return fmt.Errorf("%w: empty model", ErrDispatchPricingUnavailable)
	}
	models := in.Candidates
	if len(models) == 0 {
		models = dispatchPricingCandidates(in.Account, in.Model)
	}
	for _, model := range models {
		if dispatchModelHasPricing(ctx, in, model) {
			return nil
		}
	}
	return fmt.Errorf("%w: model=%s", ErrDispatchPricingUnavailable, strings.TrimSpace(in.Model))
}

// CheckDispatchPricingForRequest is the common dispatch hook used immediately
// before an upstream request. It reads the authenticated API key from the
// request context, preserves subscription/simple-mode semantics, and evaluates
// the selected account's mapped model.
func CheckDispatchPricingForRequest(ctx context.Context, c ContextValueGetter, account *Account, model string, kind PricingPreflightKind, resolver *ModelPricingResolver, billing *BillingService) error {
	if c == nil {
		return nil
	}
	v, ok := c.Get("api_key")
	if !ok {
		return nil
	}
	apiKey, ok := v.(*APIKey)
	if !ok || apiKey == nil {
		return nil
	}
	if inBillingSimpleMode(apiKey, billing) {
		return nil
	}
	var groupID *int64
	if apiKey.GroupID != nil {
		gid := *apiKey.GroupID
		groupID = &gid
	}
	err := CheckDispatchPricing(ctx, PricingPreflightInput{
		Resolver:   resolver,
		Billing:    billing,
		Group:      apiKey.Group,
		GroupID:    groupID,
		Account:    account,
		Model:      model,
		Kind:       kind,
		Candidates: dispatchBillingCandidates(ctx, resolver, apiKey, account, model),
	})
	return writeDispatchPricingError(c, err)
}

// A local refusal must be returned before upstream I/O and must not look like an
// upstream fault. The handlers also recognize the sentinel to skip health penalties.
func writeDispatchPricingError(c ContextValueGetter, err error) error {
	if err == nil {
		return nil
	}
	if gc, ok := c.(*gin.Context); ok && gc != nil {
		MarkOpsClientBusinessLimited(gc, OpsClientBusinessLimitedReasonLocalModelConfiguration)
		payload := gin.H{"type": "error", "error": gin.H{"type": "pricing_unavailable", "code": "pricing_unavailable", "message": "Model pricing is unavailable for this request. Please contact the administrator."}}
		if !gc.Writer.Written() {
			gc.JSON(http.StatusServiceUnavailable, payload)
		} else if strings.Contains(gc.Writer.Header().Get("Content-Type"), "text/event-stream") {
			encoded, _ := json.Marshal(payload)
			_, _ = gc.Writer.WriteString("event: error\ndata: " + string(encoded) + "\n\n")
			gc.Writer.Flush()
		}
	}
	return err
}

func dispatchBillingCandidates(ctx context.Context, resolver *ModelPricingResolver, apiKey *APIKey, account *Account, model string, alternates ...string) []string {
	original := strings.TrimSpace(model)
	if public, ok := RequestedPublicModelFromContext(ctx); ok {
		original = public
	}
	channelModel := model
	source := ""
	if resolver != nil && resolver.channelService != nil && apiKey.GroupID != nil {
		mapping := resolver.channelService.ResolveChannelMapping(ctx, *apiKey.GroupID, original)
		source = mapping.BillingModelSource
		if mapping.Mapped {
			channelModel = mapping.MappedModel
		}
	}
	mapped := model
	if account != nil {
		mapped = account.GetMappedModel(channelModel)
	}
	primary := mapped
	switch source {
	case BillingModelSourceRequested:
		primary = original
	case BillingModelSourceChannelMapped:
		primary = channelModel
	}
	all := []string{mapped, channelModel, original, model}
	all = append(all, alternates...)
	return usageBillingModelCandidates(primary, all...)
}

func inBillingSimpleMode(apiKey *APIKey, billing *BillingService) bool {
	if billing == nil || billing.cfg == nil || billing.cfg.RunMode != config.RunModeSimple {
		return false
	}
	return !billing.cfg.SimpleModeKeyRateLimitEnabled || apiKey == nil || !apiKey.HasRateLimits()
}

// CheckDispatchPricing exposes the same guard to handlers whose forwarding
// service is a protocol adapter without its own billing dependencies.
func (s *GatewayService) CheckDispatchPricing(ctx context.Context, c ContextValueGetter, account *Account, model string, kind PricingPreflightKind) error {
	if s == nil {
		return nil
	}
	return CheckDispatchPricingForRequest(ctx, c, account, model, kind, s.resolver, s.billingService)
}

// checkOpenAIDispatchPricing uses the same candidate construction and CN
// provider filter as usage billing. This prevents a CN account accepting a
// Claude-shaped request from accidentally borrowing a Claude catalog price.
func (s *OpenAIGatewayService) checkOpenAIDispatchPricing(ctx context.Context, c ContextValueGetter, account *Account, model string, alternates ...string) error {
	// Invalid campaign configuration is a fail-closed admission error. This is
	// checked before model pricing so HTTP and every WS turn cannot forward a
	// request that would later be unable to settle deterministically.
	if err := tokenCampaignPricingError(ctx); err != nil {
		return writeDispatchPricingError(c, err)
	}
	if c == nil {
		return nil
	}
	v, ok := c.Get("api_key")
	if !ok {
		return nil
	}
	apiKey, ok := v.(*APIKey)
	if !ok || apiKey == nil {
		return nil
	}
	if s == nil || inBillingSimpleMode(apiKey, s.billingService) {
		return nil
	}
	var groupID *int64
	if apiKey.GroupID != nil {
		gid := *apiKey.GroupID
		groupID = &gid
	}
	candidates := dispatchBillingCandidates(ctx, s.resolver, apiKey, account, model, alternates...)
	candidates = s.filterCNProviderBillingModelCandidates(ctx, account, apiKey, candidates)
	for _, candidate := range candidates {
		if dispatchModelHasPricing(ctx, PricingPreflightInput{
			Resolver: s.resolver,
			Billing:  s.billingService,
			Group:    apiKey.Group,
			GroupID:  groupID,
			Model:    candidate,
			Kind:     PricingPreflightToken,
		}, candidate) {
			return nil
		}
	}
	return writeDispatchPricingError(c, fmt.Errorf("%w: model=%s", ErrDispatchPricingUnavailable, strings.TrimSpace(model)))
}

// CheckDispatchPricing is the handler-facing WS/HTTP hook.
func (s *OpenAIGatewayService) CheckDispatchPricing(ctx context.Context, c ContextValueGetter, account *Account, model string, alternates ...string) error {
	return s.checkOpenAIDispatchPricing(ctx, c, account, model, alternates...)
}

type dispatchAPIKeyGetter struct{ key *APIKey }

func (g dispatchAPIKeyGetter) Get(name string) (any, bool) {
	return g.key, name == "api_key" && g.key != nil
}

// WebSocket turns use their refreshed auth snapshot and their own public name;
// neither the first frame's key nor its model can authorize a later turn.
func (s *OpenAIGatewayService) CheckDispatchPricingForAPIKey(ctx context.Context, key *APIKey, account *Account, model string, alternates ...string) error {
	ctx = context.WithValue(ctx, ctxkey.RequestedPublicModel, model)
	return s.checkOpenAIDispatchPricing(ctx, dispatchAPIKeyGetter{key}, account, model, alternates...)
}

func dispatchPricingCandidates(account *Account, requested string) []string {
	requested = strings.TrimSpace(requested)
	if account == nil {
		return []string{requested}
	}
	mapped, matched := account.ResolveMappedModel(requested)
	mapped = strings.TrimSpace(mapped)
	if !matched || mapped == "" || strings.EqualFold(mapped, requested) {
		return []string{requested}
	}
	// Include both sides. Channel requested-model billing may use the request,
	// while upstream-model billing uses the account mapping.
	return []string{requested, mapped}
}

func dispatchModelHasPricing(ctx context.Context, in PricingPreflightInput, model string) bool {
	if in.Resolver != nil {
		resolved := in.Resolver.Resolve(ctx, PricingInput{Model: model, GroupID: in.GroupID, Group: in.Group})
		if resolved != nil {
			switch in.Kind {
			case PricingPreflightToken:
				if resolved.Mode == BillingModeToken || resolved.Mode == "" {
					return tokenPricingConfigured(in.Resolver, resolved)
				}
				return (resolved.Mode == BillingModePerRequest || resolved.Mode == BillingModeImage || resolved.Mode == BillingModeVideo) && requestPricingConfigured(resolved)
			case PricingPreflightPerRequest:
				return resolved.Mode == BillingModePerRequest && requestPricingConfigured(resolved) || in.Billing != nil
			case PricingPreflightImage:
				if (resolved.Source == PricingSourceGroup || resolved.Source == PricingSourceChannel) && (resolved.Mode == BillingModeToken || resolved.Mode == "") {
					return resolved.BasePricing != nil
				}
				if resolved.Mode == BillingModeImage || resolved.Mode == BillingModePerRequest {
					return requestPricingConfigured(resolved)
				}
				return in.Billing != nil
			case PricingPreflightVideo:
				if (resolved.Source == PricingSourceGroup || resolved.Source == PricingSourceChannel) && (resolved.Mode == BillingModeToken || resolved.Mode == "") {
					return resolved.BasePricing != nil
				}
				if resolved.Mode == BillingModeVideo {
					return requestPricingConfigured(resolved)
				}
				return in.Billing != nil
			case PricingPreflightAudio:
				if resolved.Mode == BillingModePerRequest {
					return requestPricingConfigured(resolved)
				}
				return in.Billing != nil
			}
		}
	}
	if in.Billing == nil {
		return false
	}
	// Token paths may use the dynamic catalog or built-in fallback directly.
	if in.Kind == PricingPreflightToken {
		pricing, err := in.Billing.GetModelPricing(model)
		return err == nil && pricing != nil
	}
	return false
}

func tokenPricingConfigured(resolver *ModelPricingResolver, resolved *ResolvedPricing) bool {
	if resolved.BasePricing != nil {
		return true
	}
	if resolver == nil || len(resolved.Intervals) == 0 {
		return false
	}
	// Before the response its exact context size is unknown. Interval-only cards
	// must cover every possible context, including gaps and the final open range.
	points := []int{1}
	for _, iv := range resolved.Intervals {
		points = append(points, iv.MinTokens+1)
		if iv.MaxTokens != nil {
			points = append(points, *iv.MaxTokens+1)
		}
	}
	for _, point := range points {
		if resolver.GetIntervalPricing(resolved, point) == nil {
			return false
		}
	}
	return true
}

func requestPricingConfigured(resolved *ResolvedPricing) bool {
	if resolved == nil {
		return false
	}
	if resolved.channelPricing != nil && resolved.channelPricing.PerRequestPrice != nil {
		return true
	}
	if resolved.DefaultPerRequestPrice != 0 {
		return true
	}
	points := []int{1}
	for _, tier := range resolved.RequestTiers {
		points = append(points, tier.MinTokens+1)
		if tier.MaxTokens != nil {
			points = append(points, *tier.MaxTokens+1)
		}
	}
	for _, point := range points {
		tier := FindMatchingInterval(resolved.RequestTiers, point)
		if tier == nil || tier.PerRequestPrice == nil {
			return false
		}
	}
	return true
}
