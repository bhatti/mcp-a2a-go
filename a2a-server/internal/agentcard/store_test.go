package agentcard

import (
	"context"
	"testing"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStore(t *testing.T) {
	store := NewStore()

	assert.NotNil(t, store)
	assert.NotNil(t, store.cards)
}

func TestStore_Register(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	card := protocol.NewAgentCard("agent-1", "Test Agent", "1.0.0", "A test agent")
	card.AddCapability(protocol.Capability{
		Name:        "test",
		Description: "Test capability",
	})

	err := store.Register(ctx, card)
	require.NoError(t, err)

	// Verify card was registered
	retrieved, err := store.Get(ctx, "agent-1")
	require.NoError(t, err)
	assert.Equal(t, card.ID, retrieved.ID)
	assert.Equal(t, card.Name, retrieved.Name)
	assert.Len(t, retrieved.Capabilities, 1)
}

func TestStore_Register_Duplicate(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	card := protocol.NewAgentCard("agent-1", "Test Agent", "1.0.0", "Test")

	err := store.Register(ctx, card)
	require.NoError(t, err)

	// Try to register again - should fail
	err = store.Register(ctx, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestStore_Get(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	card := protocol.NewAgentCard("agent-1", "Test Agent", "1.0.0", "Test")
	store.Register(ctx, card)

	// Get existing card
	retrieved, err := store.Get(ctx, "agent-1")
	require.NoError(t, err)
	assert.Equal(t, card.ID, retrieved.ID)

	// Get non-existent card
	_, err = store.Get(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Update(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	card := protocol.NewAgentCard("agent-1", "Test Agent", "1.0.0", "Test")
	store.Register(ctx, card)

	// Update card
	card.Version = "2.0.0"
	card.AddCapability(protocol.Capability{
		Name:        "new_capability",
		Description: "New capability",
	})

	err := store.Update(ctx, card)
	require.NoError(t, err)

	// Verify update
	retrieved, err := store.Get(ctx, "agent-1")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", retrieved.Version)
	assert.Len(t, retrieved.Capabilities, 1)
}

func TestStore_Update_NotFound(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	card := protocol.NewAgentCard("non-existent", "Test", "1.0.0", "Test")

	err := store.Update(ctx, card)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Delete(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	card := protocol.NewAgentCard("agent-1", "Test Agent", "1.0.0", "Test")
	store.Register(ctx, card)

	// Delete card
	err := store.Delete(ctx, "agent-1")
	require.NoError(t, err)

	// Verify deleted
	_, err = store.Get(ctx, "agent-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_Delete_NotFound(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	err := store.Delete(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestStore_List(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Register multiple cards
	card1 := protocol.NewAgentCard("agent-1", "Agent 1", "1.0.0", "Test 1")
	card2 := protocol.NewAgentCard("agent-2", "Agent 2", "1.0.0", "Test 2")
	card3 := protocol.NewAgentCard("agent-3", "Agent 3", "1.0.0", "Test 3")

	store.Register(ctx, card1)
	store.Register(ctx, card2)
	store.Register(ctx, card3)

	// List all cards
	cards := store.List(ctx)
	assert.Len(t, cards, 3)
}

func TestStore_FindByCapability(t *testing.T) {
	store := NewStore()
	ctx := context.Background()

	// Create cards with different capabilities
	card1 := protocol.NewAgentCard("agent-1", "Agent 1", "1.0.0", "Test 1")
	card1.AddCapability(protocol.Capability{Name: "search"})
	card1.AddCapability(protocol.Capability{Name: "analyze"})

	card2 := protocol.NewAgentCard("agent-2", "Agent 2", "1.0.0", "Test 2")
	card2.AddCapability(protocol.Capability{Name: "search"})

	card3 := protocol.NewAgentCard("agent-3", "Agent 3", "1.0.0", "Test 3")
	card3.AddCapability(protocol.Capability{Name: "summarize"})

	store.Register(ctx, card1)
	store.Register(ctx, card2)
	store.Register(ctx, card3)

	// Find agents with "search" capability
	cards := store.FindByCapability(ctx, "search")
	assert.Len(t, cards, 2)

	// Find agents with "summarize" capability
	cards = store.FindByCapability(ctx, "summarize")
	assert.Len(t, cards, 1)
	assert.Equal(t, "agent-3", cards[0].ID)

	// Find agents with non-existent capability
	cards = store.FindByCapability(ctx, "non-existent")
	assert.Empty(t, cards)
}
