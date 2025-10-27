package main

import (
	"sync"
)

// In-Memoty Deck
//
// This deck could use a distributed database,
// on real implementations.
type Deck struct {
	mu    sync.RWMutex
	cards map[int]Card
}

func NewDeck() *Deck {
	return &Deck{
		cards: make(map[int]Card),
	}
}

func (deck *Deck) Add(card Card) {
	deck.mu.Lock()
	defer deck.mu.Unlock()

	deck.cards[card.ID] = card
}

func (deck *Deck) Remove(card_id int) {
	deck.mu.Lock()
	defer deck.mu.Unlock()

	delete(deck.cards, card_id)
}

func (deck *Deck) List() []Card {
	deck.mu.RLock()
	defer deck.mu.RUnlock()

	result := make([]Card, 0, len(deck.cards))

	for _, card := range deck.cards {
		result = append(result, card)
	}
	return result
}
