package cost

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Usage represents token usage and cost for a single operation
type Usage struct {
	UserID           string    `json:"user_id"`
	TaskID           string    `json:"task_id"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	Timestamp        time.Time `json:"timestamp"`
}

// Tracker tracks token usage and costs
type Tracker struct {
	mu    sync.RWMutex
	usage []Usage
}

// NewTracker creates a new cost tracker
func NewTracker() *Tracker {
	return &Tracker{
		usage: make([]Usage, 0),
	}
}

// RecordUsage records token usage and cost
func (t *Tracker) RecordUsage(ctx context.Context, usage Usage) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if usage.Timestamp.IsZero() {
		usage.Timestamp = time.Now()
	}

	t.usage = append(t.usage, usage)
	return nil
}

// GetUsage retrieves usage records for a user within a time range
func (t *Tracker) GetUsage(ctx context.Context, userID string, start, end time.Time) ([]Usage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []Usage
	for _, u := range t.usage {
		if u.UserID == userID &&
			(u.Timestamp.Equal(start) || u.Timestamp.After(start)) &&
			(u.Timestamp.Equal(end) || u.Timestamp.Before(end)) {
			result = append(result, u)
		}
	}

	return result, nil
}

// GetTotalCost calculates total cost for a user within a time range
func (t *Tracker) GetTotalCost(ctx context.Context, userID string, start, end time.Time) (float64, error) {
	usage, err := t.GetUsage(ctx, userID, start, end)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, u := range usage {
		total += u.CostUSD
	}

	return total, nil
}

// GetTotalTokens calculates total tokens for a user within a time range
func (t *Tracker) GetTotalTokens(ctx context.Context, userID string, start, end time.Time) (int, error) {
	usage, err := t.GetUsage(ctx, userID, start, end)
	if err != nil {
		return 0, err
	}

	var total int
	for _, u := range usage {
		total += u.TotalTokens
	}

	return total, nil
}

// Budget represents a user's budget constraints
type Budget struct {
	UserID          string    `json:"user_id"`
	MonthlyLimitUSD float64   `json:"monthly_limit_usd"`
	CurrentSpendUSD float64   `json:"current_spend_usd"`
	ResetAt         time.Time `json:"reset_at"`
}

// CheckBudget checks if a cost is within budget
func (b *Budget) CheckBudget(costUSD float64) bool {
	return b.CurrentSpendUSD+costUSD <= b.MonthlyLimitUSD
}

// RemainingBudget returns the remaining budget
func (b *Budget) RemainingBudget() float64 {
	remaining := b.MonthlyLimitUSD - b.CurrentSpendUSD
	if remaining < 0 {
		return 0
	}
	return remaining
}

// PercentUsed returns the percentage of budget used
func (b *Budget) PercentUsed() float64 {
	if b.MonthlyLimitUSD == 0 {
		return 0
	}
	return (b.CurrentSpendUSD / b.MonthlyLimitUSD) * 100
}

// UpdateSpend updates the current spend
func (b *Budget) UpdateSpend(costUSD float64) {
	b.CurrentSpendUSD += costUSD
}

// BudgetManager manages user budgets
type BudgetManager struct {
	mu      sync.RWMutex
	budgets map[string]*Budget
}

// NewBudgetManager creates a new budget manager
func NewBudgetManager() *BudgetManager {
	return &BudgetManager{
		budgets: make(map[string]*Budget),
	}
}

// SetBudget sets a user's budget
func (bm *BudgetManager) SetBudget(ctx context.Context, userID string, monthlyLimitUSD float64) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.budgets[userID] = &Budget{
		UserID:          userID,
		MonthlyLimitUSD: monthlyLimitUSD,
		CurrentSpendUSD: 0,
		ResetAt:         time.Now().AddDate(0, 1, 0),
	}

	return nil
}

// GetBudget retrieves a user's budget
func (bm *BudgetManager) GetBudget(ctx context.Context, userID string) (*Budget, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	budget, exists := bm.budgets[userID]
	if !exists {
		return nil, fmt.Errorf("budget for user %s not found", userID)
	}

	return budget, nil
}

// CheckAndUpdate checks if cost is within budget and updates if allowed
func (bm *BudgetManager) CheckAndUpdate(ctx context.Context, userID string, costUSD float64) (bool, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	budget, exists := bm.budgets[userID]
	if !exists {
		return false, fmt.Errorf("budget for user %s not found", userID)
	}

	if !budget.CheckBudget(costUSD) {
		return false, nil
	}

	budget.UpdateSpend(costUSD)
	return true, nil
}

// ResetBudget resets a user's current spend
func (bm *BudgetManager) ResetBudget(ctx context.Context, userID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	budget, exists := bm.budgets[userID]
	if !exists {
		return fmt.Errorf("budget for user %s not found", userID)
	}

	budget.CurrentSpendUSD = 0
	budget.ResetAt = time.Now().AddDate(0, 1, 0)
	return nil
}

// Model pricing (per 1K tokens) - based on OpenAI pricing as of 2024
var modelPricing = map[string]struct {
	PromptCost     float64
	CompletionCost float64
}{
	"gpt-4": {
		PromptCost:     0.03,
		CompletionCost: 0.06,
	},
	"gpt-4-turbo": {
		PromptCost:     0.01,
		CompletionCost: 0.03,
	},
	"gpt-3.5-turbo": {
		PromptCost:     0.0015,
		CompletionCost: 0.002,
	},
	"claude-3-opus": {
		PromptCost:     0.015,
		CompletionCost: 0.075,
	},
	"claude-3-sonnet": {
		PromptCost:     0.003,
		CompletionCost: 0.015,
	},
}

// CalculateCost calculates the cost based on model and token usage
func CalculateCost(model string, promptTokens, completionTokens int) float64 {
	pricing, exists := modelPricing[model]
	if !exists {
		// Default to gpt-3.5-turbo pricing
		pricing = modelPricing["gpt-3.5-turbo"]
	}

	promptCost := float64(promptTokens) * pricing.PromptCost / 1000.0
	completionCost := float64(completionTokens) * pricing.CompletionCost / 1000.0

	return promptCost + completionCost
}
