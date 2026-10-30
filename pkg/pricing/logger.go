package pricing

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// PriceLogger handles transaction logging for pricing auditing
type PriceLogger struct {
	transactions []CostBreakdown
	maxSize      int
	logFile      *os.File
	outputFormat string
}

// NewPriceLogger creates a new price logger with optional log file
func NewPriceLogger(maxSize int, logFilePath string, outputFormat string) (*PriceLogger, error) {
	var logFile *os.File
	var err error

	if logFilePath != "" {
		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	return &PriceLogger{
		transactions: make([]CostBreakdown, 0, maxSize),
		maxSize:      maxSize,
		logFile:      logFile,
		outputFormat: outputFormat,
	}, nil
}

// LogTransaction logs a pricing transaction
func (l *PriceLogger) LogTransaction(cost CostBreakdown) {
	if len(l.transactions) >= l.maxSize {
		// Remove oldest transaction (simple FIFO)
		l.transactions = l.transactions[1:]
	}

	l.transactions = append(l.transactions, cost)

	// Write to log file if available
	if l.logFile != nil {
		// Create log entry
		logEntry := LogEntry{
			Timestamp:  cost.Timestamp,
			Tokens:     cost.Tokens,
			Cost:       cost.TotalCost,
			CacheHit:   cost.CacheHit,
			Enterprise: cost.Enterprise,
			Currency:   cost.Currency,
		}

		// Convert to formatted string
		logLine := l.formatLogEntry(logEntry)

		if _, err := l.logFile.WriteString(logLine); err != nil {
			log.Printf("Failed to write to log file: %v", err)
		}
	}
}

// LogEntry represents a single log entry for pricing transactions
type LogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Tokens     int       `json:"tokens"`
	Cost       float64   `json:"cost"`
	CacheHit   bool      `json:"cache_hit"`
	Enterprise bool      `json:"enterprise"`
	Currency   string    `json:"currency"`
}

// formatLogEntry formats a log entry for the specified output format
func (l *PriceLogger) formatLogEntry(entry LogEntry) string {
	switch l.outputFormat {
	case "json":
		data, _ := json.MarshalIndent(entry, "", "  ")
		return string(data) + "\n"
	case "csv":
		return fmt.Sprintf("%s,%d,%.4f,%t,%t,%s\n",
			entry.Timestamp.Format(time.RFC3339), entry.Tokens, entry.Cost, entry.CacheHit, entry.Enterprise, entry.Currency)
	default: // "text"
		return fmt.Sprintf("[%s] Tokens: %d, Cost: $%.4f, Cache: %t, Enterprise: %t, Currency: %s\n",
			entry.Timestamp.Format(time.RFC3339), entry.Tokens, entry.Cost, entry.CacheHit, entry.Enterprise, entry.Currency)
	}
}

// GetAllTransactions returns all recorded transactions
func (l *PriceLogger) GetAllTransactions() []CostBreakdown {
	// Return a copy to avoid external modification
	result := make([]CostBreakdown, len(l.transactions))
	copy(result, l.transactions)
	return result
}

// ExportTransactions exports all transactions as JSON
func (l *PriceLogger) ExportTransactions() []CostBreakdown {
	return l.GetAllTransactions()
}

// Close closes the log file
func (l *PriceLogger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// GetStatistics returns pricing statistics
func (l *PriceLogger) GetStatistics() PricingStatistics {
	stats := PricingStatistics{}

	if len(l.transactions) == 0 {
		return stats
	}

	totalRevenue := 0.0
	cacheHits := 0
	enterpriseTrans := 0

	for _, trans := range l.transactions {
		totalRevenue += trans.TotalCost
		if trans.CacheHit {
			cacheHits++
		}
		if trans.Enterprise {
			enterpriseTrans++
		}
	}

	stats.TotalTransactions = len(l.transactions)
	stats.TotalRevenue = totalRevenue
	stats.AverageCost = totalRevenue / float64(len(l.transactions))
	stats.CacheHitRate = float64(cacheHits) / float64(len(l.transactions))
	stats.EnterpriseDiscountRate = float64(enterpriseTrans) / float64(len(l.transactions))

	// Find min and max costs
	if len(l.transactions) > 0 {
		minCost := l.transactions[0].TotalCost
		maxCost := l.transactions[0].TotalCost

		for _, trans := range l.transactions {
			if trans.TotalCost < minCost {
				minCost = trans.TotalCost
			}
			if trans.TotalCost > maxCost {
				maxCost = trans.TotalCost
			}
		}

		stats.MinCost = minCost
		stats.MaxCost = maxCost
	}

	return stats
}

// PricingStatistics holds pricing statistics
type PricingStatistics struct {
	TotalTransactions      int     `json:"total_transactions"`
	TotalRevenue           float64 `json:"total_revenue"`
	AverageCost            float64 `json:"average_cost"`
	MinCost                float64 `json:"min_cost"`
	MaxCost                float64 `json:"max_cost"`
	CacheHitRate           float64 `json:"cache_hit_rate"`
	EnterpriseDiscountRate float64 `json:"enterprise_discount_rate"`
}
