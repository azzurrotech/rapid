package pricing

import (
    "fmt"
    "time"
)

// PricingCalculator handles cost computation for RAPID operations
type PricingCalculator struct {
    baseCostPerToken     float64
    baseCostPerOperation float64
    ollamaPricing        bool
    cacheHitMultiplier   float64
    enterpriseDiscount   float64
}

// NewPricingCalculator creates a new pricing calculator
func NewPricingCalculator(enterprise bool) *PricingCalculator {
    calc := &PricingCalculator{
        baseCostPerToken:     0.002,    // $0.002 per 1K tokens (estimated)
        baseCostPerOperation: 0.50,     // $0.50 per operation
        ollamaPricing:        true,     // Using local Ollama models
        cacheHitMultiplier:   0.1,       // 90% discount for cache hits
        enterpriseDiscount:  0.2,       // 20% enterprise discount
    }

    if enterprise {
        calc.enterpriseDiscount = 0.3 // 30% for enterprise customers
    }

    return calc
}

// CalculateCost computes the cost for a single operation
func (p *PricingCalculator) CalculateCost(
    tokensUsed int,
    cacheHit bool,
    operationType string,
    enterprise bool,
    timestamp time.Time,
) CostBreakdown {
    // Base token cost (local Ollama is cheaper than cloud)
    tokenCost := float64(tokensUsed) / 1000 * p.baseCostPerToken

    // Operation cost
    operationCost := p.baseCostPerOperation

    // Apply cache discount if cache hit
    if cacheHit {
        tokenCost *= p.cacheHitMultiplier
        operationCost *= p.cacheHitMultiplier
    }

    // Apply enterprise discount
    if enterprise {
        tokenCost *= (1 - p.enterpriseDiscount)
        operationCost *= (1 - p.enterpriseDiscount)
    }

    // Time-based pricing (premium during business hours)
    hour := timestamp.Hour()
    if hour >= 9 && hour <= 17 {
        tokenCost *= 1.1 // 10% premium during business hours
    }

    totalCost := tokenCost + operationCost

    return CostBreakdown{
        Tokens:          tokensUsed,
        TokenCost:       tokenCost,
        OperationCost:   operationCost,
        TotalCost:       totalCost,
        Currency:        "USD",
        CacheHit:        cacheHit,
        Enterprise:      enterprise,
        Timestamp:       timestamp,
    }
}

// CostBreakdown represents the breakdown of costs for an operation
type CostBreakdown struct {
    Tokens         int       `json:"tokens"`
    TokenCost      float64   `json:"token_cost"`
    OperationCost  float64   `json:"operation_cost"`
    TotalCost      float64   `json:"total_cost"`
    Currency       string    `json:"currency"`
    CacheHit       bool      `json:"cache_hit"`
    Enterprise     bool      `json:"enterprise"`
    Timestamp      time.Time `json:"timestamp"`
}

// FormatCost formats the cost breakdown for display
func (c *CostBreakdown) FormatCost() string {
    return fmt.Sprintf("$%.4f (%.0f tokens: $%.4f + %s operation: $%.4f)",
        c.TotalCost, float64(c.Tokens),
        c.TokenCost, c.getOperationType(), c.OperationCost)
}

func (c *CostBreakdown) getOperationType() string {
    if c.CacheHit {
        return "cache"
    }
    return "full"
}

// PriceLogger handles transaction logging for pricing auditing
type PriceLogger struct {
    transactions []CostBreakdown
    maxSize      int
}

// NewPriceLogger creates a new price logger
func NewPriceLogger(maxSize int) *PriceLogger {
    return &PriceLogger{
        transactions: make([]CostBreakdown, 0, maxSize),
        maxSize:      maxSize,
    }
}

// LogTransaction logs a pricing transaction
func (l *PriceLogger) LogTransaction(cost CostBreakdown) {
    if len(l.transactions) >= l.maxSize {
        // Remove oldest transaction (simple FIFO)
        l.transactions = l.transactions[1:]
    }

    l.transactions = append(l.transactions, cost)
}

// GetTotalRevenue calculates total revenue from all logged transactions
func (l *PriceLogger) GetTotalRevenue() float64 {
    total := 0.0
    for _, trans := range l.transactions {
        total += trans.TotalCost
    }
    return total
}

// GetAverageCost calculates average cost per transaction
func (l *PriceLogger) GetAverageCost() float64 {
    if len(l.transactions) == 0 {
        return 0
    }

    total := l.GetTotalRevenue()
    return total / float64(len(l.transactions))
}

// GetTransactionsByDate returns transactions filtered by date
func (l *PriceLogger) GetTransactionsByDate(startDate, endDate time.Time) []CostBreakdown {
    var filtered []CostBreakdown
    for _, trans := range l.transactions {
        if (trans.Timestamp.After(startDate) || trans.Timestamp.Equal(startDate)) &&
            (trans.Timestamp.Before(endDate) || trans.Timestamp.Equal(endDate)) {
            filtered = append(filtered, trans)
        }
    }
    return filtered
}

// GetCacheHitRate calculates the percentage of cache hits
func (l *PriceLogger) GetCacheHitRate() float64 {
    if len(l.transactions) == 0 {
        return 0
    }

    cacheHits := 0
    for _, trans := range l.transactions {
        if trans.CacheHit {
            cacheHits++
        }
    }

    return float64(cacheHits) / float64(len(l.transactions))
}

// GetEnterpriseDiscountRate calculates the percentage of enterprise transactions
func (l *PriceLogger) GetEnterpriseDiscountRate() float64 {
    if len(l.transactions) == 0 {
        return 0
    }

    enterpriseTrans := 0
    for _, trans := range l.transactions {
        if trans.Enterprise {
            enterpriseTrans++
        }
    }

    return float64(enterpriseTrans) / float64(len(l.transactions))
}

// ExportTransactions exports all transactions as JSON
func (l *PriceLogger) ExportTransactions() []CostBreakdown {
    // Return a copy to avoid external modification
    result := make([]CostBreakdown, len(l.transactions))
    copy(result, l.transactions)
    return result
}