package service

import (
	"fmt"
	"os"
)

// Protocol-only fixtures still need the pricing dependency required by an
// authenticated dispatch. Keep their transport/configuration intact and use
// the bundled catalog rather than weakening the production guard.
func withTestGatewayPricing(s *OpenAIGatewayService) *OpenAIGatewayService {
	if s.billingService == nil {
		data, err := os.ReadFile("../../resources/model-pricing/model_prices_and_context_window.json")
		if err != nil {
			panic(fmt.Errorf("read protocol-test pricing fixture: %w", err))
		}
		pricing := &PricingService{}
		pricing.pricingData, err = pricing.parsePricingData(data)
		if err != nil {
			panic(fmt.Errorf("parse protocol-test pricing fixture: %w", err))
		}
		s.billingService = NewBillingService(s.cfg, pricing)
	}
	if s.resolver == nil {
		s.resolver = NewModelPricingResolver(s.channelService, s.billingService)
	}
	return s
}
