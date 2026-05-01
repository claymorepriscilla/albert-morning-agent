package news

import (
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

const maxAge = 24 * time.Hour

// FetchRSS fetches up to limit headlines published within the last 24 hours from an RSS feed URL.
func FetchRSS(url string, limit int) (string, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return "", fmt.Errorf("parse RSS feed: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	var lines []string
	for _, item := range feed.Items {
		if len(lines) >= limit {
			break
		}
		// Skip articles with unknown publish date or older than 24 hours.
		if item.PublishedParsed == nil || item.PublishedParsed.Before(cutoff) {
			continue
		}
		lines = append(lines, "- "+item.Title)
	}

	if len(lines) == 0 {
		return "ไม่พบข่าว", nil
	}

	return strings.Join(lines, "\n"), nil
}
