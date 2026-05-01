package news_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/claymorepriscilla/albert-morning-agent/internal/news"
)

func rssBody(titles ...string) string {
	pubDate := time.Now().UTC().Format(time.RFC1123Z)
	var items strings.Builder
	for _, title := range titles {
		fmt.Fprintf(&items, "<item><title>%s</title><pubDate>%s</pubDate></item>", title, pubDate)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Test</title>%s</channel></rss>`, items.String())
}

func TestFetchRSS(t *testing.T) {
	tests := []struct {
		name         string
		handler      http.HandlerFunc
		limit        int
		wantErr      bool
		wantContains string
		wantLines    int
	}{
		{
			name: "success — returns up to limit headlines",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, rssBody("Head 1", "Head 2", "Head 3"))
			},
			limit:     2,
			wantLines: 2,
		},
		{
			name: "success — limit larger than item count",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, rssBody("Head 1", "Head 2"))
			},
			limit:     10,
			wantLines: 2,
		},
		{
			name: "success — single item",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, rssBody("Only News"))
			},
			limit:        1,
			wantContains: "Only News",
		},
		{
			name: "empty feed returns ไม่พบข่าว",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `<?xml version="1.0"?><rss version="2.0"><channel><title>Empty</title></channel></rss>`)
			},
			limit:        5,
			wantContains: "ไม่พบข่าว",
		},
		{
			name: "limit zero returns ไม่พบข่าว",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, rssBody("Head 1"))
			},
			limit:        0,
			wantContains: "ไม่พบข่าว",
		},
		{
			name: "http 500 returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "server error", http.StatusInternalServerError)
			},
			limit:   5,
			wantErr: true,
		},
		{
			name: "invalid xml returns error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "this is not xml")
			},
			limit:   5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			result, err := news.FetchRSS(srv.URL, tt.limit)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantContains != "" {
				if !strings.Contains(result, tt.wantContains) {
					t.Errorf("result %q does not contain %q", result, tt.wantContains)
				}
				return
			}
			lines := strings.Split(strings.TrimSpace(result), "\n")
			if len(lines) != tt.wantLines {
				t.Errorf("got %d lines, want %d\nresult: %q", len(lines), tt.wantLines, result)
			}
		})
	}
}

// TestFetchRSS_Race verifies no data races when called concurrently.
// Run with: go test -race ./internal/news/...
func TestFetchRSS_Race(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, rssBody("News A", "News B", "News C"))
	}))
	defer srv.Close()

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			news.FetchRSS(srv.URL, 5) //nolint:errcheck
		}()
	}
	wg.Wait()
}
