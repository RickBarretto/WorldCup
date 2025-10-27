# Decks Service

This service manages the player's decks.

Overview
- Leader election: the node with the highest numeric `-id` is the leader.
- Leader handles mutating operations and asynchronously replicates them to peers via POST /replicate.
- Followers forward mutating requests to the leader; GET requests are served locally from each node's deck store.

## API Endpoints

There are two kinds of decks:

- Global deck: Global Storage that generates and rewards players.
- Per-user deck: User owned deck management.

Per-user API:

- `GET /users/:user/cards`
    - List cards for `:user`
- `POST /users/:user/cards`
    - Add a card for `:user` (JSON: `{"id":123,"name":"ace"}`)
- `DELETE /users/:user/cards/:id`
    - Remove card `:id` from `:user`'s deck

Global Deck API:

- `GET /cards`
    - List cards from the global deck
- `POST /cards`
    - Add a card to the global deck
- `DELETE /cards/{id}`
    - Remove a card from the global deck

- `POST /replicate`
    - Internal endpoint for replication (peers only)
- `GET /status`
    - Node status and current leader

Some of those endpoints just returns values and others proxies the leader node. But for the user the behavior would be the same for any node.

## Real Usage

### Build

```sh
go build -o decks.exe ./decks
```

### Starting

- **Node 1**
```sh
./decks.exe -id=1 -addr=http://localhost:8001 -peers=1=http://localhost:8001,2=http://localhost:8002,3=http://localhost:8003
```

- **Node 2**
```sh
./decks.exe -id=2 -addr=http://localhost:8002 -peers=1=http://localhost:8001,2=http://localhost:8002,3=http://localhost:8003
```

- **Node 3**
```sh
./decks.exe -id=3 -addr=http://localhost:8003 -peers=1=http://localhost:8001,2=http://localhost:8002,3=http://localhost:8003
```

### CURL Usage

Per-user examples:

Add a card for user `john`:

```sh
curl -X POST http://localhost:8001/users/john/cards -H "Content-Type: application/json" -d '{"id":101,"name":"Ace"}'
```

List john's cards:

```sh
curl http://localhost:8002/users/john/cards
```

Delete a card for john:

```sh
curl -X DELETE http://localhost:8002/users/john/cards/101 -v
```

Global deck examples:

Add to global deck:

```sh
curl -X POST http://localhost:8001/cards -H "Content-Type: application/json" -d '{"id":201,"name":"King"}'
```

List global deck:

```sh
curl http://localhost:8003/cards
```
