# PBL Session Summaries

## Week 1 — 23/09/2025

**Facts**

* The game must move to a distributed system with multiple servers.
* No single point of failure — the system should keep running even if one server fails.
* Use **Docker** for servers and clients.
* **Server–Server:** REST API.
* **Server–Client:** Publisher–Subscriber model.
* Card packs must be managed fairly and without duplication.
* Matches should work across servers (1×1 duels).
* The system must be tested for concurrency and failures.

**Ideas**

* Each game server runs in Docker, handles local matches, connects to clients (pub-sub), and communicates with other servers (REST).
* Pub-Sub: topics for matches, lobby, and inventory/trade.
* REST: endpoints for matchmaking, syncing, and heartbeat.
* Matchmaking: handshake + confirmation, fallback if a server fails.
* Fault tolerance: heartbeats, retries, two-phase commit, and replaying events.

**Questions**

* What is a REST API?
* What is pub-sub and which tech should we use (WebSockets, MQTT, NATS)?
* How do clients log in and join matches?
* How do we identify matches (IDs)?
* What is the message format (JSON)?
* What is idempotency and how do we avoid duplicates?
* How to avoid giving the same card pack twice?
* What happens if a server crashes mid-match?
* How do servers find each other?
* What logs do we need?

**Goals**

* Study the **Byzantine Generals Problem** (consensus and fault tolerance).
* Learn and apply **pub-sub** for Server–Client communication. (Development)
* Define topics, message format, and events.

Here’s a concise, consistent summary for **Session 2**, following the same format and tone as your Week 1 entry:

---

## Week 2 — 29/09/2025

**Facts**

* **Inter-server communication** remains essential for matches between players connected to different servers.

**Ideas**

* Use MQTT for client–server interactions thought pub-sub pattern.
* **Fail Over** mechanism implementation
* Design a **distributed consistency model** for the global state of cards to prevent duplication or loss.
* Introduce a **leader election mechanism** (e.g., Bully, Raft, or Paxos) for coordination and fault management.
* Explore distributed concurrency control (locks, transactions, eventual consistency).

**Questions**

* How to implement **leader election** effectively in a distributed environment?
* How to manage **race conditions and consistency** across multiple servers?
* How does **MQTT architecture** work in detail (broker, publisher, subscriber, topics, QoS)?
* Should we keep the current **TCP/HTTP layer** or migrate fully to MQTT for certain communications?

**Goals**

* Research and propose **data consistency mechanisms** for distributed card state management.
* **Implement inter-server communication** to synchronize matches and game state.
* (Optional) Create a **proof of concept** demonstrating full communication flow between two clients routed via MQTT brokers and intermediate servers.

---

## Week 3 — 09/10/2025

**Ideas**

* Host server to manage the match between two servers
* Token-based mutual exclusion to manage matches and decks
    * Use a middleware to make sure only peers with token can process the request
* Raft to elect the leader to manage the global card stock
* Redis as message broker
* Reset the settings when some peer goes down

**Questions**

* What should be tested?

**Goals**

* Architectural diagram of the system
* Apply publisher-subscriber for client-server pattern