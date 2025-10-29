package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)


var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (server *Server) upgradeWebsocket() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		player := request.URL.Query().Get("player_id")

		if player == "" {
			http.Error(writer, "player_id required as query param", http.StatusBadRequest)
			return
		}

		websocket, err := upgrader.Upgrade(writer, request, nil)

		if err != nil {
			log.Println("ws upgrade:", err)
			return
		}

		connection := newPlayerConnection(websocket)
		server.LinkPlayer(player, connection)

		defer func() {
			server.UnlinkPlayer(player)
			websocket.Close()
		}()

		connection.sendJSON(map[string]any{
			"type":      "welcome",
			"player_id": player,
			"server":    server.address,
		})

		for {
			_, _, err := websocket.NextReader()
			if err != nil {
				if err == io.EOF {
					return
				}
				log.Println("ws read err:", err)
				return
			}
		}
	}
}

// playMatch accepts a player's play request (5 cards) and tries to match
// / Create or enter in a match
// /
// / Request Body Format:
// / 	{
// /			"player_id": string,
// /			"cards": {
// /				"id": string,
// /				"name": string,
// /				"power": int
// /			}
// / 	}
func (server *Server) playMatch() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

		var data Challenger
		if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
			http.Error(writer, "bad json", http.StatusBadRequest)
			return
		}
		// Require a valid player id in the request body.
		if data.PlayerID == "" {
			http.Error(writer, "player_id required in request body", http.StatusBadRequest)
			return
		}
		if len(data.Cards) != 5 {
			http.Error(writer, "must send exactly 5 cards", http.StatusBadRequest)
			return
		}

		challenger := Challenger{
			PlayerID: data.PlayerID,
			Cards:    data.Cards,
		}

		// Disallow a player already in the waiting queue from playing again.
		if server.IsWaiting(challenger.PlayerID) {
			http.Error(writer, "player already queued for a match", http.StatusConflict)
			return
		}

		// try local match
		if match, ok := server.tryLocalMatch(challenger); ok {
			server.notifyLocal(match.Host.ID, map[string]interface{}{"type": "match_start", "match": match})
			server.notifyLocal(match.Guest.ID, map[string]interface{}{"type": "match_start", "match": match})
			writer.Header().Set("content-type", "application/json")
			json.NewEncoder(writer).Encode(match)
			return
		}

		// try peers: ask each peer if they have a waiter
		callbackURL := fmt.Sprintf("http://%s/start-remote-match", server.address)
		tried := false
		for _, p := range server.ListPeers() {
			tried = true
			body := map[string]interface{}{"player_id": data.PlayerID, "cards": data.Cards, "callback": callbackURL, "server": server.address}
			b, _ := json.Marshal(body)
			resp, err := http.Post(fmt.Sprintf("http://%s/find-waiter", p), "application/json", bytes.NewReader(b))
			if err != nil {
				log.Printf("error contacting peer %s: %v", p, err)
				continue
			}
			if resp.StatusCode == http.StatusNoContent {
				continue
			}
			var match Match
			if err := json.NewDecoder(resp.Body).Decode(&match); err == nil {
				server.notifyLocal(data.PlayerID, map[string]interface{}{
					"type":  "match_start",
					"match": match,
				})
				writer.Header().Set("content-type", "application/json")
				json.NewEncoder(writer).Encode(match)
				return
			}
		}

		server.enqueueWaiter(challenger)
		writer.WriteHeader(http.StatusAccepted)

		if !tried {
			writer.Write([]byte("queued local; no peers configured"))
		} else {
			writer.Write([]byte("queued local; no peer match found"))
		}
	}

}

// / Notify the challenger
func (server *Server) startRemoteMatch() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var match Match

		if err := json.NewDecoder(request.Body).Decode(&match); err != nil {
			http.Error(writer, "bad json", http.StatusBadRequest)
			return
		}

		if match.Host.Server == server.address {
			server.notifyLocal(match.Host.ID, map[string]any{
				"type":  "match_start",
				"match": match,
			})
		}

		if match.Guest.Server == server.address {
			server.notifyLocal(match.Guest.ID, map[string]any{
				"type":  "match_start",
				"match": match,
			})
		}

		writer.WriteHeader(http.StatusOK)
	}
}

// / Endpoint for add or list Peers
func (server *Server) managePeers() http.HandlerFunc {
	return func(response http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			json.NewEncoder(response).Encode(server.ListPeers())
		case http.MethodPost:
			var req struct {
				Peer string `json:"peer"`
			}
			if err := json.NewDecoder(request.Body).Decode(&req); err != nil {
				http.Error(
					response,
					"bad json",
					http.StatusBadRequest,
				)
				return
			}
			server.AddPeer(req.Peer)
			response.WriteHeader(http.StatusCreated)
		default:
			http.Error(
				response,
				"method not allowed",
				http.StatusMethodNotAllowed,
			)
		}
	}
}
