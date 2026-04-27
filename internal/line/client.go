package line

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	pushURL      = "https://api.line.me/v2/bot/message/push"
	broadcastURL = "https://api.line.me/v2/bot/message/broadcast"
)

// Client sends messages via LINE Messaging API.
// When broadcast is true, messages are sent to all users who added the bot.
type Client struct {
	accessToken string
	userID      string
	broadcast   bool
	http        *http.Client
}

// NewClient creates a LINE message client.
// Set broadcast=true to send to all users who added the bot instead of a single user.
func NewClient(accessToken, userID string, broadcast bool) *Client {
	return &Client{
		accessToken: accessToken,
		userID:      userID,
		broadcast:   broadcast,
		http:        &http.Client{},
	}
}

// Send delivers a plain-text message.
// In broadcast mode it sends to all users; otherwise it pushes to the configured user.
func (c *Client) Send(text string) error {
	var (
		endpoint string
		payload  map[string]any
	)

	if c.broadcast {
		endpoint = broadcastURL
		payload = map[string]any{
			"messages": []map[string]string{
				{"type": "text", "text": text},
			},
		}
	} else {
		endpoint = pushURL
		payload = map[string]any{
			"to": c.userID,
			"messages": []map[string]string{
				{"type": "text", "text": text},
			},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LINE API status %d", resp.StatusCode)
	}

	return nil
}
