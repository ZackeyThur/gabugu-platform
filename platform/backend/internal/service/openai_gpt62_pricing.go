package service

import "strings"

// GPT-6.2 Sol / Astra billing identity.
//
// Production groups already route gpt-6.2-sol / gpt-6.2-astra (e.g. group 38
// "AZ GPT"), but neither the upstream price catalog nor this source priced them,
// so the pre-dispatch check failed closed with 503 pricing_unavailable.
//
// This file only decides *which fallback card* a GPT-6.2 request name maps to.
// It deliberately does not touch normalizeKnownOpenAICodexModel: that function
// also rewrites upstream model names for the Codex transform, and GPT-6.2 must
// keep being forwarded under its own name.
const (
	openAIGPT62Sol   = "gpt-6.2-sol"
	openAIGPT62Astra = "gpt-6.2-astra"
)

// openAIGPT62FallbackKey returns the fallback-card key for a GPT-6.2 request
// name, or "" for anything else. Provider prefixes, case, and the local
// effort / compact / date suffixes already accepted for the GPT-6.1 family are
// recognised. Bare "gpt-6.2", other tiers (terra, luna, ...) and other
// versions are NOT matched, so unknown neighbours keep failing closed.
func openAIGPT62FallbackKey(model string) string {
	canonical := canonicalizeOpenAIModelAliasSpelling(model)
	if canonical == "" {
		return ""
	}
	for _, base := range []string{openAIGPT62Sol, openAIGPT62Astra} {
		if canonical == base {
			return base
		}
		if suffix, ok := strings.CutPrefix(canonical, base+"-"); ok && isOpenAIGPT62Suffix(suffix) {
			return base
		}
	}
	return ""
}

func isOpenAIGPT62Suffix(suffix string) bool {
	switch suffix {
	case "none", "minimal", "low", "medium", "high", "xhigh", "max", "openai-compact":
		return true
	}
	if len(suffix) != 8 {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// openAIBillingModelKey identifies the price card without rewriting the model
// sent upstream. Dated GPT-6.2 aliases stay intact on the wire, but billing and
// channel fallback use their published Sol/Astra card.
func openAIBillingModelKey(model string) string {
	if normalized := normalizeKnownOpenAICodexModel(model); normalized != "" {
		return normalized
	}
	return openAIGPT62FallbackKey(model)
}
