package news

import (
	"fmt"
	"strings"

	"github.com/mmcdole/gofeed"
)

// FetchRSS fetches up to limit headlines from an RSS feed URL.
func FetchRSS(url string, limit int) (string, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)
	if err != nil {
		return "", fmt.Errorf("parse RSS feed: %w", err)
	}

	var lines []string
	for i, item := range feed.Items {
		if i >= limit {
			break
		}
		lines = append(lines, "- "+item.Title)
	}

	if len(lines) == 0 {
		return "ไม่พบข่าว", nil
	}

	return strings.Join(lines, "\n"), nil
}
