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
		baseCostPerToken:     0.002, // $0.002 per 1K tokens (estimated)
		baseCostPerOperation: 0.50,  // $0.50 per operation
		ollamaPricing:        true,  // Using local Ollama models
		cacheHitMultiplier:   0.1,   // 90% discount for cache hits
		enterpriseDiscount:   0.2,   // 20% enterprise discount
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
		Tokens:        tokensUsed,
		TokenCost:     tokenCost,
		OperationCost: operationCost,
		TotalCost:     totalCost,
		Currency:      "USD",
		CacheHit:      cacheHit,
		Enterprise:    enterprise,
		Timestamp:     timestamp,
	}
}

// CostBreakdown represents the breakdown of costs for an operation
type CostBreakdown struct {
	Tokens        int       `json:"tokens"`
	TokenCost     float64   `json:"token_cost"`
	OperationCost float64   `json:"operation_cost"`
	TotalCost     float64   `json:"total_cost"`
	Currency      string    `json:"currency"`
	CacheHit      bool      `json:"cache_hit"`
	Enterprise    bool      `json:"enterprise"`
	Timestamp     time.Time `json:"timestamp"`
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
