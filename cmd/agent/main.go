package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/claymorepriscilla/albert-morning-agent/internal/config"
	"github.com/claymorepriscilla/albert-morning-agent/internal/gemini"
	"github.com/claymorepriscilla/albert-morning-agent/internal/gold"
	"github.com/claymorepriscilla/albert-morning-agent/internal/line"
	"github.com/claymorepriscilla/albert-morning-agent/internal/news"
)

const rssLimit = 10

type briefing struct {
	emoji  string
	label  string
	topic  string
	rssURL string
}

var dailyBriefings = []briefing{
	{
		emoji:  "🤖",
		label:  "ข่าว AI",
		topic:  "AI และเทคโนโลยี",
		rssURL: "https://news.google.com/rss/search?q=artificial+intelligence+AI+when:1d&hl=th&gl=TH&ceid=TH:th",
	},
	{
		emoji:  "🇹🇭",
		label:  "หุ้นไทย",
		topic:  "หุ้นไทย",
		rssURL: "https://news.google.com/rss/search?q=%E0%B8%95%E0%B8%A5%E0%B8%B2%E0%B8%94%E0%B8%AB%E0%B8%A5%E0%B8%B1%E0%B8%81%E0%B8%97%E0%B8%A3%E0%B8%B1%E0%B8%9E%E0%B8%A2%E0%B9%8C+SET+%E0%B8%AB%E0%B8%B8%E0%B9%89%E0%B8%99%E0%B9%84%E0%B8%97%E0%B8%A2+when:1d&hl=th&gl=TH&ceid=TH:th",
	},
	{
		emoji:  "🇺🇸",
		label:  "หุ้นอเมริกา",
		topic:  "หุ้นอเมริกา",
		rssURL: "https://news.google.com/rss/search?q=stock+market+nasdaq+S%26P500+when:6h&hl=en-US&gl=US&ceid=US:en",
	},
}

var thaiMonths = [13]string{
	"", "มกราคม", "กุมภาพันธ์", "มีนาคม", "เมษายน", "พฤษภาคม", "มิถุนายน",
	"กรกฎาคม", "สิงหาคม", "กันยายน", "ตุลาคม", "พฤศจิกายน", "ธันวาคม",
}

// lotteryRSSURL returns a Google News RSS URL scoped to the nearest Thai lottery draw date
// (1st or 16th of the month) to avoid picking up results from previous draws.
func lotteryRSSURL(t time.Time) string {
	drawDay := 1
	if t.Day() >= 16 {
		drawDay = 16
	}
	q := fmt.Sprintf("ผลสลากกินแบ่งรัฐบาล %d %s %d", drawDay, thaiMonths[t.Month()], t.Year()+543)
	v := url.Values{"q": {q}, "hl": {"th"}, "gl": {"TH"}, "ceid": {"TH:th"}}
	return "https://news.google.com/rss/search?" + v.Encode()
}

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	geminiClient, err := gemini.NewClient(ctx, cfg.GroqAPIKey)
	if err != nil {
		log.Fatalf("gemini: %v", err)
	}
	defer geminiClient.Close()

	lineClient := line.NewClient(cfg.LineAccessToken, cfg.LineUserID, cfg.Broadcast)
	now := time.Now().In(time.FixedZone("Asia/Bangkok", 7*60*60))
	today := now.Format("02/01/2006")

	if os.Getenv("LOTTERY_ONLY") == "true" {
		// Afternoon run — check lottery results only, URL scoped to the current draw date.
		lotteryBriefing := briefing{emoji: "🎰", label: "ผลหวยไทย", topic: "หวยไทย", rssURL: lotteryRSSURL(now)}
		processIfNewsFound(ctx, geminiClient, lineClient, lotteryBriefing, today)
	} else {
		for _, b := range dailyBriefings {
			process(ctx, geminiClient, lineClient, b, today)
		}
		processGold(ctx, geminiClient, lineClient, today)
	}

	log.Println("Morning Agent completed.")
}

// processGold fetches Thai gold prices + news summary and sends them as a single message.
func processGold(ctx context.Context, g *gemini.Client, l *line.Client, today string) {
	log.Printf("[start] ราคาทองคำ")

	price, err := gold.FetchPrice()
	if err != nil {
		log.Printf("[skip]  ราคาทองคำ — fetch price: %v", err)
		return
	}

	var summary string
	headlines, err := news.FetchRSS(
		"https://news.google.com/rss/search?q=gold+price+thailand+%E0%B8%97%E0%B8%AD%E0%B8%87%E0%B8%84%E0%B8%B3+when:1d&hl=th&gl=TH&ceid=TH:th",
		rssLimit,
	)
	switch {
	case err != nil:
		log.Printf("[warn]  ราคาทองคำ — fetch news: %v (sending price only)", err)
	case headlines == "ไม่พบข่าว":
		log.Printf("[warn]  ราคาทองคำ — no recent news, sending price only")
	default:
		summary, err = g.Summarize("ราคาทองคำและแนวโน้ม", headlines, today)
		if err != nil {
			log.Printf("[warn]  ราคาทองคำ — summarize: %v (sending price only)", err)
		}
	}

	msg := gold.FormatMessage(price, summary, today)

	if err := l.Send(msg); err != nil {
		log.Printf("[skip]  ราคาทองคำ — LINE: %v", err)
		return
	}

	log.Printf("[done]  ราคาทองคำ")
}

// processIfNewsFound is like process but skips silently when no recent headlines are found.
// Used for event-driven briefings (e.g. lottery) where no news means the event hasn't happened yet.
func processIfNewsFound(ctx context.Context, g *gemini.Client, l *line.Client, b briefing, today string) {
	log.Printf("[start] %s", b.label)

	headlines, err := news.FetchRSS(b.rssURL, rssLimit)
	if err != nil {
		log.Printf("[skip]  %s — fetch: %v", b.label, err)
		return
	}
	if headlines == "ไม่พบข่าว" {
		log.Printf("[skip]  %s — no recent news found", b.label)
		return
	}

	summary, err := g.Summarize(b.topic, headlines, today)
	if err != nil {
		log.Printf("[skip]  %s — summarize: %v", b.label, err)
		return
	}

	msg := fmt.Sprintf("%s *%s* ประจำวัน %s\n\n%s", b.emoji, b.label, today, summary)

	if err := l.Send(msg); err != nil {
		log.Printf("[skip]  %s — LINE: %v", b.label, err)
		return
	}

	log.Printf("[done]  %s", b.label)
}

// process fetches, summarises, and sends one briefing.
// Errors are logged and skipped (best-effort).
func process(ctx context.Context, g *gemini.Client, l *line.Client, b briefing, today string) {
	log.Printf("[start] %s", b.label)

	headlines, err := news.FetchRSS(b.rssURL, rssLimit)
	if err != nil {
		log.Printf("[skip]  %s — fetch: %v", b.label, err)
		return
	}
	if headlines == "ไม่พบข่าว" {
		log.Printf("[skip]  %s — no recent news found", b.label)
		return
	}

	summary, err := g.Summarize(b.topic, headlines, today)
	if err != nil {
		log.Printf("[skip]  %s — summarize: %v", b.label, err)
		return
	}

	msg := fmt.Sprintf("%s *%s* ประจำวัน %s\n\n%s", b.emoji, b.label, today, summary)

	if err := l.Send(msg); err != nil {
		log.Printf("[skip]  %s — LINE: %v", b.label, err)
		return
	}

	log.Printf("[done]  %s", b.label)
}
