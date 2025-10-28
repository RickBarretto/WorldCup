Match server

Simple distributed 1v1 card-match server.

- Start server: go run ./match -port=8081 -peers=localhost:8082,localhost:8083

Endpoints:
- GET /ws?player_id=<id> : open websocket to receive events
- POST /play : {"player_id":"alice","cards":[{id,name,power},...]} (must be 5 cards)
- POST /find-waiter : used by peers (internal)
- POST /start-remote-match : used by peers to notify of match (internal)
- GET/POST /peers : list/add peers

Frontend:
- A simple web UI is included under `match/frontend/`. When the server runs it will serve the UI at `/`.
- Open `http://localhost:8081/` in your browser (replace port as appropriate). The UI connects a websocket and lets you submit 5 cards.

Protocol:
- Each player connects to their server via websocket and then POSTs /play with 5 cards.
- Server will try to match locally; if none, it will ask peers via /find-waiter.
- When match is created both players receive a websocket message with type "match_start" and the match payload including winner (sum of card power).

This is a minimal demo and uses in-memory state only.
