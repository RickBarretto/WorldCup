package main

type Address = string
type CardID = string
type MatchID = string
type Username = string
type CardPower = int

type Host = PlayerInfo
type Guest = PlayerInfo

type Card struct {
	ID    CardID    `json:"id"`
	Name  string    `json:"name"`
	Power CardPower `json:"power"`
}

type PlayerInfo struct {
	ID     Username `json:"player_id"`
	Server Address  `json:"server"`
	Cards  []Card   `json:"cards"`
}

type Match struct {
	ID     MatchID  `json:"id"`
	Host   Host     `json:"p1"`
	Guest  Guest    `json:"p2"`
	Winner Username `json:"winner"`
}

type WaitingPlayer struct {
	PlayerID Username
	Cards    []Card
}

type Challenger = WaitingPlayer
