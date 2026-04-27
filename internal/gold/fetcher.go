package gold

import (
	"encoding/json"
	"fmt"
	"net/http"
)

var apiURL = "https://api.chnwt.dev/thai-gold-api/latest"

// Price holds buy/sell prices for gold bar and gold jewelry.
type Price struct {
	BarBuy      string
	BarSell     string
	OrnamentBuy string
	OrnamentSell string
}

type apiResponse struct {
	Status   string `json:"status"`
	Response struct {
		Price struct {
			Gold struct {
				Buy  string `json:"buy"`
				Sell string `json:"sell"`
			} `json:"gold"`
			GoldBar struct {
				Buy  string `json:"buy"`
				Sell string `json:"sell"`
			} `json:"gold_bar"`
		} `json:"price"`
	} `json:"response"`
}

// FetchPrice retrieves the latest Thai gold prices from the Gold Traders Association.
func FetchPrice() (*Price, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("fetch gold price: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gold api status %d", resp.StatusCode)
	}

	var result apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode gold price: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("gold api returned status: %s", result.Status)
	}

	return &Price{
		BarBuy:       result.Response.Price.GoldBar.Buy,
		BarSell:      result.Response.Price.GoldBar.Sell,
		OrnamentBuy:  result.Response.Price.Gold.Buy,
		OrnamentSell: result.Response.Price.Gold.Sell,
	}, nil
}

// FormatMessage combines gold price and AI summary into a single LINE message.
func FormatMessage(p *Price, summary, date string) string {
	base := fmt.Sprintf(
		"🥇 *ราคาทองคำ* %s\n\nทองคำแท่ง\n  รับซื้อ: %s บาท\n  ขายออก: %s บาท\n\nทองรูปพรรณ\n  รับซื้อ: %s บาท\n  ขายออก: %s บาท",
		date,
		p.BarBuy, p.BarSell,
		p.OrnamentBuy, p.OrnamentSell,
	)
	if summary == "" {
		return base
	}
	return base + "\n\n" + summary
}
