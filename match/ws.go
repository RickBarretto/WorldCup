package main

import "bytes"
import "encoding/json"
import "flag"
import "fmt"
import "io"
import "log"
import "net/http"
import "sync"
import "time"

import "github.com/gorilla/websocket"

/// Player Websocket Connection
type PlayerConnection struct {
	connection *websocket.Conn
	mutex      sync.Mutex
}

func newPlayerConnection(connection *websocket.Conn) *PlayerConnection {
	return &PlayerConnection{connection: connection}
}

func (player *PlayerConnection) sendJSON(value any) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if player.connection == nil {
		return
	}
	timeout := time.Now().Add(5 * time.Second)
	player.connection.SetWriteDeadline(timeout)
	_ = player.connection.WriteJSON(value)
}

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

		// send welcome
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
/// Create or enter in a match
///
/// Request Body Format:
/// 	{
///			"player_id": string,
///			"cards": {
///				"id": string,
///				"name": string,
///				"power": int
///			}
/// 	}
func (server *Server) playMatch() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {

		var data Challenger
		if err := json.NewDecoder(request.Body).Decode(&data); err != nil {
			http.Error(writer, "bad json", http.StatusBadRequest)
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

/// Notify the challenger
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

/// Endpoint for add or list Peers
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

func StartServer(address Address, peers []Address) {
	server := NewServer(address)
	for _, p := range peers {
		if p != "" {
			server.AddPeer(p)
		}
	}

	// -- API --
	http.HandleFunc("/ws", server.upgradeWebsocket())
	http.HandleFunc("/play", server.playMatch())
	http.HandleFunc("/find-waiter", server.FindWaiter)
	http.HandleFunc("/start-remote-match", server.startRemoteMatch())
	http.HandleFunc("/peers", server.managePeers())

	// -- Frontend --
	fs := http.FileServer(http.Dir("./match/frontend"))
	http.Handle("/", fs)

	log.Printf("match server listening on %s\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func main() {
	var port string
	var rawPeers string

	flag.StringVar(&port, "port", "8081", "server listen port")
	flag.StringVar(&rawPeers, "peers", "", "comma-separated peer host:port list")
	flag.Parse()

	address := fmt.Sprintf("localhost:%s", port)
	peers := []Address{}

	if rawPeers != "" {
		peers = append(peers, splitComma(rawPeers)...)
	}

	StartServer(address, peers)
}

func splitComma(s string) []string {
	out := []string{}
	cur := ""

	for _, ch := range s {
		if ch == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(ch)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
