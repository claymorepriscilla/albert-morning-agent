package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/claymorepriscilla/albert-morning-agent/internal/config"
	"github.com/claymorepriscilla/albert-morning-agent/internal/gemini"
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

	for _, b := range dailyBriefings {
		process(ctx, geminiClient, lineClient, b, today)
	}

	if d := time.Now().Day(); d == 1 || d == 16 {
		process(ctx, geminiClient, lineClient, lotteryBriefing, today)
	}

	log.Println("Morning Agent completed.")
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

	summary, err := g.Summarize(b.topic, headlines)
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
