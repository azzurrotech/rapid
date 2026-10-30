package pricing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCalculateCostAppliesCacheAndEnterprise(t *testing.T) {
	calc := NewPricingCalculator(false)
	ts := time.Date(2026, 1, 2, 20, 0, 0, 0, time.UTC) // outside business hours

	full := calc.CalculateCost(10_000, false, "analysis", false, ts)
	cached := calc.CalculateCost(10_000, true, "analysis", false, ts)
	if cached.TotalCost >= full.TotalCost {
		t.Errorf("cache hit did not reduce cost: %+v vs %+v", cached, full)
	}
	enterprise := calc.CalculateCost(10_000, false, "analysis", true, ts)
	if enterprise.TotalCost >= full.TotalCost {
		t.Errorf("enterprise discount did not reduce cost: %+v vs %+v", enterprise, full)
	}
	if full.Currency != "USD" {
		t.Errorf("currency = %q", full.Currency)
	}
}

func TestPriceLoggerWritesAndSummarises(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pricing.log")
	logger, err := NewPriceLogger(3, path, "json")
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	calc := NewPricingCalculator(false)
	ts := time.Now().UTC()
	for i := 0; i < 5; i++ {
		logger.LogTransaction(calc.CalculateCost(1000, i%2 == 0, "test", false, ts))
	}

	// The ring keeps the newest three.
	if got := len(logger.GetAllTransactions()); got != 3 {
		t.Errorf("kept %d transactions, want 3", got)
	}
	stats := logger.GetStatistics()
	if stats.TotalTransactions != 3 || stats.TotalRevenue <= 0 {
		t.Errorf("stats = %+v", stats)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"cost"`) {
		t.Errorf("log is not JSON lines:\n%s", data)
	}
}
