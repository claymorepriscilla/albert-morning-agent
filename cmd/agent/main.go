package main

import (
	"context"
	"fmt"
	"log"
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
		rssURL: "https://news.google.com/rss/search?q=artificial+intelligence+AI&hl=th&gl=TH&ceid=TH:th",
	},
	{
		emoji:  "🇹🇭",
		label:  "หุ้นไทย",
		topic:  "หุ้นไทย",
		rssURL: "https://news.google.com/rss/search?q=SET+index+thailand+stock&hl=th&gl=TH&ceid=TH:th",
	},
	{
		emoji:  "🇺🇸",
		label:  "หุ้นอเมริกา",
		topic:  "หุ้นอเมริกา",
		rssURL: "https://news.google.com/rss/search?q=stock+market+nasdaq+S%26P500&hl=en&gl=US&ceid=US:en",
	},
}

var lotteryBriefing = briefing{
	emoji:  "🎰",
	label:  "ผลหวยไทย",
	topic:  "หวยไทย",
	rssURL: "https://news.google.com/rss/search?q=ผลสลากกินแบ่งรัฐบาล+หวย&hl=th&gl=TH&ceid=TH:th",
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
	today := time.Now().In(time.FixedZone("Asia/Bangkok", 7*60*60)).Format("02/01/2006")

	if os.Getenv("LOTTERY_ONLY") == "true" {
		// Afternoon run — check lottery results only (no hardcoded date, rely on 24h RSS filter).
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
		"https://news.google.com/rss/search?q=gold+price+thailand+%E0%B8%97%E0%B8%AD%E0%B8%87%E0%B8%84%E0%B8%B3&hl=th&gl=TH&ceid=TH:th",
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
