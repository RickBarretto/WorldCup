# Match Service

This service allows 1v1 matches between players.

## Overview

- Each card is a JSON object: { "id": string, "name": string, "power": int }.
- A client must submit exactly 5 cards to play a single-turn match.
- Winner is determined by the sum of card `power` values. If equal, result is `draw`.
- Servers keep in-memory state only (players, waiting queue). 
- Servers can be configured with peers so players on different servers can match.

## Real Usage

### Build

```sh
go build ./match
```

### Run (single server)

```sh
./match.exe -port=8081
```

### Run (3 peers)


- **Node 1**
```sh
./match.exe -port=8081 -peers=localhost:8082,localhost:8083
```

- **Node 2**
```sh
./match.exe -port=8082 -peers=localhost:8081,localhost:8083
```

- **Node 3**
```sh
./match.exe -port=8083 -peers=localhost:8081,localhost:8082
```

### Frontend

Open the frontend at `http://localhost:8081`, `http://localhost:8082` or `http://localhost:8083` and interact with it. 

This minimal static frontend is included under `match/frontend/` folder and is served at `/`.
The UI opens a websocket to `/ws` and posts to `/play`.

## Public HTTP API

These are the endpoints clients will use.

- **GET** `/ws?player_id=:<id>`
	- Upgrade to a WebSocket for the given player id. Server uses this connection to push events (match start, notifications).
- **POST** `/play`
	- Start a play request. Body JSON:

		```json
		{
			"player_id": "alice",
			"cards": [
				{"id":"c1","name":"A","power":3},
				{"id":"c2","name":"B","power":4},
				{"id":"c3","name":"C","power":2},
				{"id":"c4","name":"D","power":1},
				{"id":"c5","name":"E","power":5}
			]
		}
		```
	- The server requires exactly 5 cards.
    - If a local waiting player exists, the server creates a match immediately and returns the `match` JSON.
        - Otherwise, the server queries peers for waiting players. If a peer matches, the server returns the `match` JSON.
        - If no match is found, the request enqueues the player locally and returns HTTP 202 Accepted.

### Administrator API

- **GET** `/peers`
	- Returns JSON array of peer addresses configured on this server (host:port strings).
- **POST** `/peers`
	- Add a peer to the list. Body JSON: `{ "peer": "host:port" }`. Returns HTTP 201 on success.

## Internal API

- **POST** `/find-waiter`
	- A peer calls this to ask if this server currently has a waiting player. Request body JSON:
		```json
		{
			"player_id": "challenger-id",
			"cards": [{"id":"...","name":"...","power":1}, ...],
			"callback": "http://challenger-server/start-remote-match",
			"server": "challenger-server-address"
		}
		```
	- If this server has no waiting player it responds with HTTP 204 No Content.
	- If there is a waiting player, this server will create the match (pairing its waiter and the remote challenger), notify its local player over WebSocket, POST the match JSON to the provided `callback` URL on the challenger server, and respond to the caller with the `match` JSON.
- **POST** `/start-remote-match`
	- A peer calls this to notify this server that a cross-server match was created. The request body is the `match` JSON; the server will notify any local player(s) in the match over WebSocket and return HTTP 200.

## WebSocket messages

WebSocket messages are JSON objects. The UI and clients should expect at least these message types:

- Welcome (sent on connect)
	```json
	{ "type": "welcome", "player_id": "alice", "server": "localhost:8081" }
	```
- Match start (sent to both players when a match is created)
	```json
	{ "type": "match_start", "match": { "id": "<match-id>", "p1": {...}, "p2": {...}, "winner": "player-id-or-draw" } }
	```
	- `match.p1` and `match.p2` are `PlayerInfo` objects containing `player_id`, `server`, and the `cards` array.


## Match object shape


```json
{
	"id": "<match-id>",
	"p1": {"player_id":"alice", "server":"localhost:8081","cards":[ ... ]},
	"p2": {"player_id":"bob","server":"localhost:8082","cards":[ ... ]},
	"winner": "alice" | "bob" | "draw"
}
```
