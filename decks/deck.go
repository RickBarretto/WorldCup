package main

import "sync"

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

// DeckStore holds the global deck and per-user decks.
type DeckStore struct {
	mu      sync.RWMutex
	global  *Deck
	users   map[string]*Deck
}

func NewDeckStore() *DeckStore {
	return &DeckStore{
		global: NewDeck(),
		users:  make(map[string]*Deck),
	}
}

// resolveDeck returns the deck for a user; empty user -> global
func (ds *DeckStore) resolveDeck(user string) *Deck {
	if user == "" {
		return ds.global
	}

	ds.mu.Lock()
	defer ds.mu.Unlock()
	d, ok := ds.users[user]
	if !ok {
		d = NewDeck()
		ds.users[user] = d
	}
	return d
}

func (ds *DeckStore) Add(user string, card Card) {
	d := ds.resolveDeck(user)
	d.Add(card)
}

func (ds *DeckStore) Remove(user string, card_id int) {
	d := ds.resolveDeck(user)
	d.Remove(card_id)
}

func (ds *DeckStore) List(user string) []Card {
	d := ds.resolveDeck(user)
	return d.List()
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
