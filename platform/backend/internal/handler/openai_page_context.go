package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var codexSelectedPagePart = regexp.MustCompile(`^codex_apps_selected_page_part_([1-9][0-9]{0,3})_of_([1-9][0-9]{0,3})$`)

// Codex supplies companion Page context as named, unpaired function outputs.
// Keep that external content at user priority and retain its identity for the
// existing Page instructions. Real tool outputs must retain their call IDs.
func normalizeCodexUntrustedContext(body []byte) ([]byte, bool) {
	if !mayContainCodexCallOutputCandidate(body, "codex_apps_open_page", "codex_apps_selected_page_part_", "browser_info", "goal_objective", "untrusted_context") {
		return body, false
	}
	if !hasUniqueJSONMembers(body) {
		return body, false
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var request map[string]any
	if decoder.Decode(&request) != nil {
		return body, false
	}
	input, ok := request["input"].([]any)
	if !ok {
		return body, false
	}
	changed := false
	for i, raw := range input {
		item, ok := raw.(map[string]any)
		if !ok || !isCodexUntrustedContext(item) {
			continue
		}
		name := stringField(item, "name")
		output := item["output"].(string)
		input[i] = map[string]any{
			"type": "message", "role": "user",
			"content": []any{map[string]any{
				"type": "input_text",
				"text": "<external_" + name + ">" + output + "</external_" + name + ">",
			}},
		}
		changed = true
	}
	if !changed {
		return body, false
	}
	normalized, err := json.Marshal(request)
	if err != nil {
		return body, false
	}
	return normalized, true
}

func isCodexUntrustedContext(item map[string]any) bool {
	if stringField(item, "type") != "function_call_output" || stringField(item, "namespace") != "agent" {
		return false
	}
	if callID, exists := item["call_id"]; exists {
		value, ok := callID.(string)
		if !ok || strings.TrimSpace(value) != "" {
			return false
		}
	}
	output, ok := item["output"].(string)
	if !ok {
		return false
	}
	name := stringField(item, "name")
	switch name {
	case "browser_info", "goal_objective", "untrusted_context":
		return strings.TrimSpace(output) != ""
	}
	selected := name != "codex_apps_open_page"
	if selected {
		parts := codexSelectedPagePart.FindStringSubmatch(name)
		if parts == nil {
			return false
		}
		part, _ := strconv.Atoi(parts[1])
		total, _ := strconv.Atoi(parts[2])
		if part > total {
			return false
		}
		output = strings.ReplaceAll(output, "\r\n", "\n")
		header := fmt.Sprintf("Companion Page snapshot part %d of %d:\n", part, total)
		if !strings.HasPrefix(output, header) {
			return false
		}
		output = strings.TrimPrefix(output, header)
	}
	if !hasUniqueJSONMembers([]byte(output)) {
		return false
	}
	var page map[string]json.RawMessage
	if json.Unmarshal([]byte(output), &page) != nil || page == nil {
		return false
	}
	pageID, exists := page["page_id"]
	if !exists {
		return false
	}
	emptyPage := bytes.Equal(bytes.TrimSpace(pageID), []byte("null"))
	if !emptyPage {
		var id string
		if json.Unmarshal(pageID, &id) != nil || strings.TrimSpace(id) == "" {
			return false
		}
	}
	if !selected {
		return len(page) == 1
	}
	view, exists := page["document_view"]
	if !exists || len(page) != 2 {
		return false
	}
	view = bytes.TrimSpace(view)
	if bytes.Equal(view, []byte("null")) {
		return true
	}
	return !emptyPage && len(view) > 0 && view[0] == '{'
}
