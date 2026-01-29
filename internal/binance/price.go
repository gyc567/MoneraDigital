package binance

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type PriceCache struct {
	Prices    map[string]float64
	UpdatedAt time.Time
}

type PriceService struct {
	baseURL    string
	httpClient *http.Client
	cache      *PriceCache
	cacheMu    sync.RWMutex
	ticker     *time.Ticker
	stopChan   chan struct{}
}

var (
	instance *PriceService
	once     sync.Once
)

func NewPriceService() *PriceService {
	once.Do(func() {
		instance = &PriceService{
			baseURL: "https://api.binance.com",
			httpClient: &http.Client{
				Timeout: 10 * time.Second,
			},
			cache: &PriceCache{
				Prices:    make(map[string]float64),
				UpdatedAt: time.Now(),
			},
			stopChan: make(chan struct{}),
		}

		go instance.startBackgroundFetcher()
	})
	return instance
}

func (s *PriceService) startBackgroundFetcher() {
	fetchInterval := 5 * time.Minute
	s.ticker = time.NewTicker(fetchInterval)

	s.fetchAllPrices()

	for {
		select {
		case <-s.ticker.C:
			s.fetchAllPrices()
		case <-s.stopChan:
			log.Println("[PriceService] Background fetcher stopped")
			return
		}
	}
}

func (s *PriceService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
}

func (s *PriceService) fetchAllPrices() {
	symbols := []string{"BTC", "ETH", "SOL", "ADA", "XRP", "DOGE"}

	symbolStrings := make([]string, len(symbols))
	for i, sym := range symbols {
		symbolStrings[i] = "\"" + sym + "USDT\""
	}
	url := s.baseURL + "/api/v3/ticker/price?symbols=[" + strings.Join(symbolStrings, ",") + "]"

	resp, err := s.httpClient.Get(url)
	if err != nil {
		log.Printf("[PriceService] Failed to fetch prices: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[PriceService] HTTP error: %d", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[PriceService] Response body: %s", string(body))
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[PriceService] Failed to read response: %v", err)
		return
	}

	var priceList []PriceResponse
	if err := json.Unmarshal(body, &priceList); err != nil {
		log.Printf("[PriceService] Failed to parse JSON: %v", err)
		return
	}

	s.cacheMu.Lock()
	for _, price := range priceList {
		symbol := strings.TrimSuffix(price.Symbol, "USDT")
		s.cache.Prices[symbol] = parsePriceSilent(price.Price)
	}
	s.cache.Prices["USDT"] = 1.0
	s.cache.Prices["USDC"] = 1.0
	s.cache.Prices["DAI"] = 1.0
	s.cache.UpdatedAt = time.Now()
	s.cacheMu.Unlock()

	log.Printf("[PriceService] Prices updated successfully at %s", s.cache.UpdatedAt.Format("2006-01-02 15:04:05"))
}

func (s *PriceService) GetCachedPrice(currency string) (float64, bool) {
	s.cacheMu.RLock()
	price, ok := s.cache.Prices[currency]
	s.cacheMu.RUnlock()
	return price, ok
}

func (s *PriceService) GetCachedPrices() map[string]float64 {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	result := make(map[string]float64)
	for k, v := range s.cache.Prices {
		result[k] = v
	}
	return result
}

func (s *PriceService) GetLastUpdateTime() time.Time {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.cache.UpdatedAt
}

func (s *PriceService) FetchAllPricesForAPI() {
	s.fetchAllPrices()
}

func (s *PriceService) GetUSDValueFromCache(amount float64, currency string) float64 {
	if currency == "USDT" || currency == "USDC" || currency == "DAI" {
		return amount
	}

	price, ok := s.GetCachedPrice(currency)
	if !ok {
		return 0
	}

	return amount * price
}

func (s *PriceService) GetPricesFromCache(currencies []string) map[string]float64 {
	prices := make(map[string]float64)
	for _, currency := range currencies {
		price, ok := s.GetCachedPrice(currency)
		if ok {
			prices[currency] = price
		}
	}
	return prices
}

func (s *PriceService) GetSinglePrice(currency string) (float64, error) {
	url := s.baseURL + "/api/v3/ticker/price?symbol=" + currency + "USDT"
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var price PriceResponse
	if err := json.Unmarshal(body, &price); err != nil {
		return 0, err
	}

	return parsePrice(price.Price)
}

func (s *PriceService) RefreshPrice(currency string) error {
	price, err := s.GetSinglePrice(currency)
	if err != nil {
		return err
	}

	s.cacheMu.Lock()
	s.cache.Prices[currency] = price
	s.cache.UpdatedAt = time.Now()
	s.cacheMu.Unlock()

	return nil
}

func parsePrice(priceStr string) (float64, error) {
	var price float64
	_, err := fmt.Sscanf(priceStr, "%f", &price)
	return price, err
}

func parsePriceSilent(priceStr string) float64 {
	price, _ := parsePrice(priceStr)
	return price
}
