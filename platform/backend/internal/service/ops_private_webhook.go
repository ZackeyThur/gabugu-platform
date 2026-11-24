package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

type opsPrivateWebhookPayload struct {
	EventType string `json:"event_type"`
	Target    string `json:"target"`
	Title     string `json:"title,omitempty"`
	Severity  string `json:"severity,omitempty"`
	Status    string `json:"status,omitempty"`
	Text      string `json:"text"`
	SentAt    string `json:"sent_at"`
}

func sendOpsPrivateWebhook(
	ctx context.Context,
	cfg OpsPrivateWebhookConfig,
	bearerToken string,
	payload opsPrivateWebhookPayload,
) error {
	if !cfg.Enabled {
		return nil
	}
	target := strings.TrimSpace(cfg.Target)
	if target == "" {
		return errors.New("private webhook target is empty")
	}
	if isDisallowedOpsGroupTarget(target) {
		return errors.New("refusing to send ops private webhook to group chat (@chatroom)")
	}
	webhookURL := strings.TrimSpace(cfg.URL)
	if !isValidOpsHTTPURL(webhookURL) {
		return errors.New("invalid private webhook URL")
	}

	payload.Target = target
	payload.SentAt = time.Now().UTC().Format(time.RFC3339)

	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, webhookURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if tok := strings.TrimSpace(bearerToken); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		redactedErr := redactOpsErrorSnippet(err.Error(), 240)
		logger.LegacyPrintf("service.ops_private_webhook", "[OpsPrivateWebhook] send failed (event=%s): %s", payload.EventType, redactedErr)
		return fmt.Errorf("private webhook request failed: %s", redactedErr)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = fmt.Errorf("private webhook returned HTTP %d", resp.StatusCode)
		logger.LegacyPrintf("service.ops_private_webhook", "[OpsPrivateWebhook] %v (event=%s)", err, payload.EventType)
		return err
	}
	return nil
}
