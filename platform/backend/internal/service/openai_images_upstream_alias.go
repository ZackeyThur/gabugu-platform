package service

// These configured upstream H aliases use the OpenAI Images generation and editing
// contract. Accept them only after an account maps a valid public image model;
// keep public model validation, pricing and OAuth model selection unchanged.
func isOpenAIImagesUpstreamAlias(model string) bool {
	switch model {
	case "all-image-2-H", "all-image-2.5-flare-H", "all-image-2.5-sunburst-H",
		"all-image-2.6-flare-H", "all-image-2.6-sunburst-H":
		return true
	default:
		return false
	}
}
