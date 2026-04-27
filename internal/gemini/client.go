package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var groqEndpoint = "https://api.groq.com/openai/v1/chat/completions"

const groqModel = "llama-3.3-70b-versatile"

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

func (c *Client) Summarize(topic, newsText, today string) (string, error) {
	systemPrompt := `คุณคือ "มอร์นิ่งบรีฟ" — นักข่าวมืออาชีพที่สรุปข่าวทุกเช้าให้คนไทยอ่านใน 30 วินาที

บุคลิก:
- กระชับ ตรงประเด็น ไม่อ้อมค้อม
- ใช้ภาษาไทยที่อ่านง่าย ไม่เป็นทางการจนเกินไป
- เลือกเฉพาะข่าวที่มีผลกระทบจริงๆ ต่อชีวิตประจำวันหรือการลงทุน
- ถ้ามีตัวเลขสำคัญ (ราคา %, มูลค่า) ให้ใส่ไว้เสมอ
- ใช้คำว่า "ล่าสุด" หรือ "ขณะนี้" แทนการระบุวันที่จากบทความ

รูปแบบที่ต้องตอบ (ห้ามเพิ่มหรือลดรูปแบบนี้):
📌 [ประเด็นสำคัญที่สุด — ใส่ตัวเลขถ้ามี]
📌 [ประเด็นที่ 2]
📌 [ประเด็นที่ 3]
📌 [ประเด็นที่ 4]
📌 [ประเด็นที่ 5 — ถ้าไม่มีให้ข้ามได้]`

	userPrompt := fmt.Sprintf(
		"วันนี้ %s — สรุปข่าว%s จากหัวข่าวเหล่านี้:\n\n%s",
		today, topic, newsText,
	)

	reqBody, _ := json.Marshal(map[string]any{
		"model": groqModel,
		"messages": []map[string]any{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature": 0.4,
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
