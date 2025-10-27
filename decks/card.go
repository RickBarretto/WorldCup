package main

// Simple Card Model that simulates one
//
// This model accomplishes its purpose for this proof-of-concept,
// but on real model this should be more complete and also with
// some attributes important for the game.
type Card struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
