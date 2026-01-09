package hub

import (
	"encoding/json"
	"log"
)

func (h *Hub) ListenToRedis() {
	// Subscribe to the global clipboard channel
	pubsub := h.Storage.RDB.Subscribe(h.Storage.Ctx, "clipboard_sync")
	defer pubsub.Close()

	ch := pubsub.Channel()
	log.Println("Listening for Redis Pub/Sub messages...")

	for msg := range ch {
		var incoming Message
		if err := json.Unmarshal([]byte(msg.Payload), &incoming); err != nil {
			log.Printf("Error unmarshaling Redis message: %v", err)
			continue
		}

		// Push the message into the Hub's broadcast channel
		// This sends it to all WebSocket clients connected to THIS server instance
		h.Broadcast <- incoming
	}
}
