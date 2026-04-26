package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"
	groqModel    = "llama-3.3-70b-versatile"
)

type Client struct {
	apiKey string
	http   *http.Client
	ctx    context.Context
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("groq api key is empty")
	}
	return &Client{apiKey: apiKey, http: &http.Client{}, ctx: ctx}, nil
}

func (c *Client) Close() {}

func (c *Client) Summarize(topic, newsText string) (string, error) {
	prompt := fmt.Sprintf(
		"คุณคือผู้ช่วยสรุปข่าว สรุปข่าว%sต่อไปนี้เป็นภาษาไทย\nกระชับ อ่านง่าย เลือกเฉพาะประเด็นสำคัญ ไม่เกิน 5 ข้อ\n\nข่าว:\n%s\n\nรูปแบบการตอบ:\n📌 ...\n📌 ...\n📌 ...",
		topic, newsText,
	)
	reqBody, _ := json.Marshal(map[string]any{
		"model":    groqModel,
		"messages": []map[string]string{{"role": "user", "content": prompt}},
	})
	req, err := http.NewRequestWithContext(c.ctx, http.MethodPost, groqEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq api status %d: %s", resp.StatusCode, body)
	}
	var result struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from groq")
	}
	return result.Choices[0].Message.Content, nil
}
