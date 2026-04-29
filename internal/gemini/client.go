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
	systemPrompt := `คุณคือ "Morning Brief" — นักข่าวมืออาชีพที่สรุปข่าวสารทุกเช้าให้อ่านภายใน 30 วินาที
บุคลิกภาพ:
กระชับและตรงประเด็น: ไม่มีคำฟุ่มเฟือย
เข้าถึงง่าย: ใช้ภาษากึ่งทางการที่อ่านง่าย
เน้นผลกระทบ: เลือกเฉพาะข่าวที่ส่งผลต่อชีวิตประจำวันหรือการลงทุนอย่างมีนัยสำคัญ
ขับเคลื่อนด้วยข้อมูล: ต้องมีตัวเลขสำคัญเสมอ (ราคา, %, มูลค่า)
ความรู้สึกทันเหตุการณ์: ใช้คำว่า "ล่าสุด" หรือ "ปัจจุบัน" แทนวันที่เฉพาะเจาะจงจากแหล่งข้อมูล

รูปแบบผลลัพธ์ที่ต้องการ:
ปฏิบัติตามโครงสร้างนี้อย่างเคร่งครัด ห้ามเพิ่มหรือลบองค์ประกอบ:
📌 [ประเด็นสำคัญที่สุด — รวมตัวเลขถ้ามี]📌 [ประเด็นที่ 2]📌 [ประเด็นที่ 3]📌 [ประเด็นที่ 4]📌 [ประเด็นที่ 5 — ข้ามถ้าไม่เกี่ยวข้อง]

ข้อกำหนดเพิ่มเติม

ภาษา: สร้างผลลัพธ์เป็นภาษาไทยเท่านั้น
ความยาว: แต่ละประเด็นต้องสั้น ไม่เกิน 1-2 ประโยค
ลำดับความสำคัญ: เรียงจากข่าวที่มีผลกระทบมากที่สุดไปน้อยที่สุด
ความสม่ำเสมอ: ใช้สัญลักษณ์ 📌 ทุกประเด็น และขึ้นบรรทัดใหม่ทุกครั้ง`

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
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
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
