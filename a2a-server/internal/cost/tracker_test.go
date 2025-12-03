package cost

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracker(t *testing.T) {
	tracker := NewTracker()

	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.usage)
}

func TestTracker_RecordUsage(t *testing.T) {
	tracker := NewTracker()
	ctx := context.Background()

	usage := Usage{
		UserID:      "user-1",
		TaskID:      "task-123",
		Model:       "gpt-4",
		PromptTokens: 100,
		CompletionTokens: 50,
		TotalTokens: 150,
		CostUSD:     0.003,
	}

	err := tracker.RecordUsage(ctx, usage)
	require.NoError(t, err)

	// Verify usage was recorded
	total, err := tracker.GetUsage(ctx, "user-1", time.Now().Add(-1*time.Hour), time.Now().Add(1*time.Hour))
	require.NoError(t, err)
	assert.Equal(t, 1, len(total))
	assert.Equal(t, usage.TotalTokens, total[0].TotalTokens)
}

func TestTracker_GetUsage(t *testing.T) {
	tracker := NewTracker()
	ctx := context.Background()

	now := time.Now()

	// Record multiple usage entries
	usage1 := Usage{
		UserID:      "user-1",
		TaskID:      "task-1",
		Model:       "gpt-4",
		TotalTokens: 100,
		CostUSD:     0.002,
		Timestamp:   now.Add(-2 * time.Hour),
	}
	usage2 := Usage{
		UserID:      "user-1",
		TaskID:      "task-2",
		Model:       "gpt-3.5-turbo",
		TotalTokens: 200,
		CostUSD:     0.0004,
		Timestamp:   now.Add(-1 * time.Hour),
	}
	usage3 := Usage{
		UserID:      "user-2",
		TaskID:      "task-3",
		Model:       "gpt-4",
		TotalTokens: 150,
		CostUSD:     0.003,
		Timestamp:   now.Add(-30 * time.Minute),
	}

	tracker.RecordUsage(ctx, usage1)
	tracker.RecordUsage(ctx, usage2)
	tracker.RecordUsage(ctx, usage3)

	// Get usage for user-1
	usage, err := tracker.GetUsage(ctx, "user-1", now.Add(-3*time.Hour), now)
	require.NoError(t, err)
	assert.Len(t, usage, 2)

	// Get usage for user-2
	usage, err = tracker.GetUsage(ctx, "user-2", now.Add(-3*time.Hour), now)
	require.NoError(t, err)
	assert.Len(t, usage, 1)

	// Get usage with time filter
	usage, err = tracker.GetUsage(ctx, "user-1", now.Add(-90*time.Minute), now)
	require.NoError(t, err)
	assert.Len(t, usage, 1)
}

func TestTracker_GetTotalCost(t *testing.T) {
	tracker := NewTracker()
	ctx := context.Background()

	now := time.Now()

	// Record multiple usage entries
	tracker.RecordUsage(ctx, Usage{
		UserID:    "user-1",
		CostUSD:   0.002,
		Timestamp: now.Add(-1 * time.Hour),
	})
	tracker.RecordUsage(ctx, Usage{
		UserID:    "user-1",
		CostUSD:   0.003,
		Timestamp: now.Add(-30 * time.Minute),
	})
	tracker.RecordUsage(ctx, Usage{
		UserID:    "user-2",
		CostUSD:   0.005,
		Timestamp: now.Add(-15 * time.Minute),
	})

	// Get total cost for user-1
	totalCost, err := tracker.GetTotalCost(ctx, "user-1", now.Add(-2*time.Hour), now)
	require.NoError(t, err)
	assert.InDelta(t, 0.005, totalCost, 0.0001)

	// Get total cost for user-2
	totalCost, err = tracker.GetTotalCost(ctx, "user-2", now.Add(-2*time.Hour), now)
	require.NoError(t, err)
	assert.InDelta(t, 0.005, totalCost, 0.0001)
}

func TestTracker_GetTotalTokens(t *testing.T) {
	tracker := NewTracker()
	ctx := context.Background()

	now := time.Now()

	tracker.RecordUsage(ctx, Usage{
		UserID:      "user-1",
		TotalTokens: 100,
		Timestamp:   now.Add(-1 * time.Hour),
	})
	tracker.RecordUsage(ctx, Usage{
		UserID:      "user-1",
		TotalTokens: 200,
		Timestamp:   now.Add(-30 * time.Minute),
	})

	totalTokens, err := tracker.GetTotalTokens(ctx, "user-1", now.Add(-2*time.Hour), now)
	require.NoError(t, err)
	assert.Equal(t, 300, totalTokens)
}

func TestBudget_CheckBudget(t *testing.T) {
	budget := &Budget{
		UserID:        "user-1",
		MonthlyLimitUSD: 10.0,
		CurrentSpendUSD: 5.0,
	}

	// Under budget
	allowed := budget.CheckBudget(3.0)
	assert.True(t, allowed)

	// Exactly at budget
	allowed = budget.CheckBudget(5.0)
	assert.True(t, allowed)

	// Over budget
	allowed = budget.CheckBudget(6.0)
	assert.False(t, allowed)
}

func TestBudget_RemainingBudget(t *testing.T) {
	budget := &Budget{
		UserID:        "user-1",
		MonthlyLimitUSD: 10.0,
		CurrentSpendUSD: 3.5,
	}

	remaining := budget.RemainingBudget()
	assert.InDelta(t, 6.5, remaining, 0.0001)
}

func TestBudget_PercentUsed(t *testing.T) {
	budget := &Budget{
		UserID:        "user-1",
		MonthlyLimitUSD: 10.0,
		CurrentSpendUSD: 2.5,
	}

	percent := budget.PercentUsed()
	assert.InDelta(t, 25.0, percent, 0.01)

	// Test 100% used
	budget.CurrentSpendUSD = 10.0
	percent = budget.PercentUsed()
	assert.InDelta(t, 100.0, percent, 0.01)

	// Test over 100%
	budget.CurrentSpendUSD = 12.0
	percent = budget.PercentUsed()
	assert.InDelta(t, 120.0, percent, 0.01)
}

func TestBudget_UpdateSpend(t *testing.T) {
	budget := &Budget{
		UserID:        "user-1",
		MonthlyLimitUSD: 10.0,
		CurrentSpendUSD: 3.0,
	}

	budget.UpdateSpend(2.5)
	assert.InDelta(t, 5.5, budget.CurrentSpendUSD, 0.0001)
}

func TestBudgetManager_SetBudget(t *testing.T) {
	manager := NewBudgetManager()
	ctx := context.Background()

	err := manager.SetBudget(ctx, "user-1", 50.0)
	require.NoError(t, err)

	budget, err := manager.GetBudget(ctx, "user-1")
	require.NoError(t, err)
	assert.Equal(t, "user-1", budget.UserID)
	assert.Equal(t, 50.0, budget.MonthlyLimitUSD)
	assert.Equal(t, 0.0, budget.CurrentSpendUSD)
}

func TestBudgetManager_GetBudget_NotFound(t *testing.T) {
	manager := NewBudgetManager()
	ctx := context.Background()

	_, err := manager.GetBudget(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestBudgetManager_CheckAndUpdate(t *testing.T) {
	manager := NewBudgetManager()
	ctx := context.Background()

	manager.SetBudget(ctx, "user-1", 10.0)

	// First charge - should succeed
	allowed, err := manager.CheckAndUpdate(ctx, "user-1", 3.0)
	require.NoError(t, err)
	assert.True(t, allowed)

	budget, _ := manager.GetBudget(ctx, "user-1")
	assert.InDelta(t, 3.0, budget.CurrentSpendUSD, 0.0001)

	// Second charge - should succeed
	allowed, err = manager.CheckAndUpdate(ctx, "user-1", 5.0)
	require.NoError(t, err)
	assert.True(t, allowed)

	budget, _ = manager.GetBudget(ctx, "user-1")
	assert.InDelta(t, 8.0, budget.CurrentSpendUSD, 0.0001)

	// Third charge - should fail (exceeds budget)
	allowed, err = manager.CheckAndUpdate(ctx, "user-1", 5.0)
	require.NoError(t, err)
	assert.False(t, allowed)

	// Verify spend wasn't updated
	budget, _ = manager.GetBudget(ctx, "user-1")
	assert.InDelta(t, 8.0, budget.CurrentSpendUSD, 0.0001)
}

func TestBudgetManager_ResetBudget(t *testing.T) {
	manager := NewBudgetManager()
	ctx := context.Background()

	manager.SetBudget(ctx, "user-1", 10.0)
	manager.CheckAndUpdate(ctx, "user-1", 7.0)

	budget, _ := manager.GetBudget(ctx, "user-1")
	assert.Equal(t, 7.0, budget.CurrentSpendUSD)

	// Reset budget
	err := manager.ResetBudget(ctx, "user-1")
	require.NoError(t, err)

	budget, _ = manager.GetBudget(ctx, "user-1")
	assert.Equal(t, 0.0, budget.CurrentSpendUSD)
	assert.Equal(t, 10.0, budget.MonthlyLimitUSD)
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		promptTokens     int
		completionTokens int
		expectedCost     float64
	}{
		{
			name:             "gpt-4",
			model:            "gpt-4",
			promptTokens:     1000,
			completionTokens: 500,
			expectedCost:     0.06, // (1000 * 0.03 / 1000) + (500 * 0.06 / 1000) = 0.03 + 0.03 = 0.06
		},
		{
			name:             "gpt-3.5-turbo",
			model:            "gpt-3.5-turbo",
			promptTokens:     1000,
			completionTokens: 500,
			expectedCost:     0.0025, // (1000 * 0.0015 + 500 * 0.002) / 1000
		},
		{
			name:             "unknown model defaults to gpt-3.5",
			model:            "unknown",
			promptTokens:     1000,
			completionTokens: 500,
			expectedCost:     0.0025,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateCost(tt.model, tt.promptTokens, tt.completionTokens)
			assert.InDelta(t, tt.expectedCost, cost, 0.0001)
		})
	}
}
