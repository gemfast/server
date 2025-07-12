package hookshot

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"slices"

	"github.com/gemfast/server/internal/db"
	"github.com/rs/zerolog/log"
)

// TriggerEvent sends the payload to all webhooks registered for the event and records the delivery.
func TriggerEvent(database *db.DB, event string, payload any) error {
	webhooks, err := database.ListWebhooks()
	if err != nil {
		return fmt.Errorf("failed to list webhooks: %w", err)
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	for _, wh := range webhooks {
		if !slices.Contains(wh.Events, event) {
			continue
		}

		go deliverWebhook(database, wh, event, payloadBytes)
	}
	return nil
}
func deliverWebhook(database *db.DB, wh *db.Webhook, event string, payload []byte) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", wh.URL, bytes.NewReader(payload))
	if err != nil {
		recordDelivery(database, wh.ID, event, payload, 0, "failed to create request: "+err.Error(), false)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if wh.Secret != "" {
		sig := computeHMAC(payload, wh.Secret)
		req.Header.Set("X-Hub-Signature-256", "sha256="+sig)
	}
	resp, err := client.Do(req)
	if err != nil {
		recordDelivery(database, wh.ID, event, payload, 0, "failed to send: "+err.Error(), false)
		return
	}
	defer resp.Body.Close()
	respBody, _ := ioutil.ReadAll(resp.Body)
	recordDelivery(database, wh.ID, event, payload, resp.StatusCode, string(respBody), resp.StatusCode >= 200 && resp.StatusCode < 300)
}

func computeHMAC(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func recordDelivery(database *db.DB, webhookID, event string, payload []byte, status int, respBody string, success bool) {
	delivery := &db.WebhookDelivery{
		WebhookID:      webhookID,
		Event:          event,
		Payload:        payload,
		ResponseStatus: status,
		ResponseBody:   respBody,
		Success:        success,
	}
	err := database.AddWebhookDelivery(delivery)
	if err != nil {
		log.Error().Err(err).Msg("failed to record webhook delivery")
	}
}
