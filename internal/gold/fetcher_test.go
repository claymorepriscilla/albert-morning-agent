package gold_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/claymorepriscilla/albert-morning-agent/internal/gold"
)

const validGoldJSON = `{
	"status": "success",
	"response": {
		"price": {
			"gold":     {"buy": "42,718.00", "sell": "43,850.00"},
			"gold_bar": {"buy": "43,250.00", "sell": "43,350.00"}
		}
	}
}`

func TestFetchPrice(t *testing.T) {
	tests := []struct {
		name        string
		handler     http.HandlerFunc
		wantErr     bool
		wantBarBuy  string
		wantBarSell string
		wantOrnBuy  string
		wantOrnSell string
	}{
		{
			name: "success — all prices parsed correctly",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, validGoldJSON)
			},
			wantBarBuy:  "43,250.00",
			wantBarSell: "43,350.00",
			wantOrnBuy:  "42,718.00",
			wantOrnSell: "43,850.00",
		},
		{
			name: "error — http 500",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "server error", http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "error — invalid json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "{bad json}")
			},
			wantErr: true,
		},
		{
			name: "error — api status not success",
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, `{"status":"error","response":{}}`)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			gold.SetAPIURL(srv.URL)

			price, err := gold.FetchPrice()

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if price.BarBuy != tt.wantBarBuy {
				t.Errorf("BarBuy = %q, want %q", price.BarBuy, tt.wantBarBuy)
			}
			if price.BarSell != tt.wantBarSell {
				t.Errorf("BarSell = %q, want %q", price.BarSell, tt.wantBarSell)
			}
			if price.OrnamentBuy != tt.wantOrnBuy {
				t.Errorf("OrnamentBuy = %q, want %q", price.OrnamentBuy, tt.wantOrnBuy)
			}
			if price.OrnamentSell != tt.wantOrnSell {
				t.Errorf("OrnamentSell = %q, want %q", price.OrnamentSell, tt.wantOrnSell)
			}
		})
	}
}

// TestFetchPrice_ServerUnavailable covers the HTTP transport error path.
func TestFetchPrice_ServerUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	gold.SetAPIURL(srv.URL)

	_, err := gold.FetchPrice()
	if err == nil {
		t.Fatal("expected error when server is unavailable, got nil")
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name    string
		price   *gold.Price
		summary string
		date    string
		checks  []string
	}{
		{
			name: "contains all price fields and summary",
			price: &gold.Price{
				BarBuy:       "43,250",
				BarSell:      "43,350",
				OrnamentBuy:  "42,718",
				OrnamentSell: "43,850",
			},
			summary: "📌 แนวโน้มทองขึ้น",
			date:    "27/04/2026",
			checks: []string{
				"🥇",
				"27/04/2026",
				"ทองคำแท่ง",
				"43,250",
				"43,350",
				"ทองรูปพรรณ",
				"42,718",
				"43,850",
				"📌 แนวโน้มทองขึ้น",
			},
		},
		{
			name: "empty summary still formats price block",
			price: &gold.Price{
				BarBuy: "40,000", BarSell: "40,100",
				OrnamentBuy: "39,500", OrnamentSell: "40,500",
			},
			summary: "",
			date:    "01/01/2026",
			checks:  []string{"ทองคำแท่ง", "ทองรูปพรรณ", "01/01/2026"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gold.FormatMessage(tt.price, tt.summary, tt.date)
			for _, want := range tt.checks {
				if !strings.Contains(result, want) {
					t.Errorf("result missing %q\nfull result:\n%s", want, result)
				}
			}
		})
	}
}

// TestFetchPrice_Race verifies no data races on concurrent calls.
// Run with: go test -race ./internal/gold/...
func TestFetchPrice_Race(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, validGoldJSON)
	}))
	defer srv.Close()

	gold.SetAPIURL(srv.URL)

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			gold.FetchPrice() //nolint:errcheck
		}()
	}
	wg.Wait()
}
