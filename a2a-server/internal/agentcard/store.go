package agentcard

import (
	"context"
	"fmt"
	"sync"

	"github.com/bhatti/mcp-a2a-go/a2a-server/internal/protocol"
)

// Store manages agent card registration and discovery
type Store struct {
	mu    sync.RWMutex
	cards map[string]*protocol.AgentCard
}

// NewStore creates a new agent card store
func NewStore() *Store {
	return &Store{
		cards: make(map[string]*protocol.AgentCard),
	}
}

// Register registers a new agent card
func (s *Store) Register(ctx context.Context, card *protocol.AgentCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cards[card.ID]; exists {
		return fmt.Errorf("agent %s already registered", card.ID)
	}

	s.cards[card.ID] = card
	return nil
}

// Get retrieves an agent card by ID
func (s *Store) Get(ctx context.Context, id string) (*protocol.AgentCard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	card, exists := s.cards[id]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", id)
	}

	return card, nil
}

// Update updates an existing agent card
func (s *Store) Update(ctx context.Context, card *protocol.AgentCard) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cards[card.ID]; !exists {
		return fmt.Errorf("agent %s not found", card.ID)
	}

	s.cards[card.ID] = card
	return nil
}

// Delete deletes an agent card
func (s *Store) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.cards[id]; !exists {
		return fmt.Errorf("agent %s not found", id)
	}

	delete(s.cards, id)
	return nil
}

// List lists all registered agent cards
func (s *Store) List(ctx context.Context) []*protocol.AgentCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cards := make([]*protocol.AgentCard, 0, len(s.cards))
	for _, card := range s.cards {
		cards = append(cards, card)
	}

	return cards
}

// FindByCapability finds agents that have a specific capability
func (s *Store) FindByCapability(ctx context.Context, capability string) []*protocol.AgentCard {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*protocol.AgentCard
	for _, card := range s.cards {
		for _, cap := range card.Capabilities {
			if cap.Name == capability {
				result = append(result, card)
				break
			}
		}
	}

	return result
}
