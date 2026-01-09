package hub

import (
	"log"
	"sync"
	"time"

	"github.com/msniranjan18/online-clipboard/pkg/store"
)

type Message struct {
	RoomID  string `json:"room_id"`
	Content string `json:"content"`
	Action  string `json:"action"`
}

type Hub struct {
	// Storage contains our Postgres and Redis clients
	Storage *store.Store

	// Registered clients categorized by RoomID
	Rooms map[string]map[*Client]bool

	Broadcast  chan Message
	SaveQueue  chan Message // buffered channel for DB writes
	Register   chan *Client
	Unregister chan *Client
	mu         sync.Mutex
}

// NewHub now correctly accepts the storage dependency
func NewHub(s *store.Store) *Hub {
	return &Hub{
		Storage:    s,
		Rooms:      make(map[string]map[*Client]bool),
		Broadcast:  make(chan Message),
		SaveQueue:  make(chan Message, 1000), // Buffer up to 1000 writes
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {

	// Start the DB writer worker
	go h.writeWorker()

	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.Rooms[client.RoomID] == nil {
				h.Rooms[client.RoomID] = make(map[*Client]bool)
			}
			h.Rooms[client.RoomID][client] = true
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Rooms[client.RoomID][client]; ok {
				delete(h.Rooms[client.RoomID], client)
				close(client.Send)
			}
			h.mu.Unlock()

		case msg := <-h.Broadcast:
			h.mu.Lock()
			// Send the message to all clients currently in the specific room
			// Fire and Forget for the UI
			for client := range h.Rooms[msg.RoomID] {
				select {
				case client.Send <- []byte(msg.Content):
				default:
					close(client.Send)
					delete(h.Rooms[msg.RoomID], client)
				}
			}
			h.mu.Unlock()
			select {
			case h.SaveQueue <- msg:
			default:
				log.Println("Save queue full, dropping write to protect performance")
			}
		}
	}
}

func (h *Hub) writeWorker() {
	// A map to keep track of pending updates per room
	pending := make(map[string]string)
	// A timer to trigger the batch save
	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case msg := <-h.SaveQueue:
			log.Printf("msn.Action %s", msg.Action)

			switch msg.Action {
			case "CLEAR":
				// Remove from pending map so a scheduled save doesn't overwrite the delete
				delete(pending, msg.RoomID)
				// Perform immediate delete
				if err := h.Storage.DeleteContent(msg.RoomID); err != nil {
					log.Printf("Error deleting room %s: %v", msg.RoomID, err)
				}
				log.Printf("Room %s cleared from DB", msg.RoomID)

			case "SAVE":
				// Perform immediate save (Manual Override)
				delete(pending, msg.RoomID)
				if err := h.Storage.SaveContent(msg.RoomID, msg.Content); err != nil {
					log.Printf("Error forced-saving room %s: %v", msg.RoomID, err)
				}

			default:
				// Normal "UPDATE" action: just update the map for the 10s ticker
				pending[msg.RoomID] = msg.Content
			}

		case <-ticker.C:
			if len(pending) == 0 {
				continue // Don't do anything if there's no new data
			}
			// Every 2 seconds, flush everything in the map to the DB
			for roomID, content := range pending {
				err := h.Storage.SaveContent(roomID, content)
				if err != nil {
					log.Printf("Debounced save error for %s: %v", roomID, err)
					continue
				}
				// Remove from map once saved
				delete(pending, roomID)
			}
		}
	}
}
