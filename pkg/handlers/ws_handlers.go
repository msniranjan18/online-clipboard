package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/msniranjan18/online-clipboard/pkg/hub"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true }, // Relax for dev
}

func HandleWS(h *hub.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract RoomID from URL (e.g., /ws/my-room)
		// segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/ws/"), "/")
		// roomID := segments[0]
		pathParts := strings.Split(r.URL.Path, "/")
		roomID := pathParts[len(pathParts)-1]

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Upgrade error: %v", err)
			return
		}

		client := &hub.Client{
			Hub:    h,
			Conn:   conn,
			RoomID: roomID,
			Send:   make(chan []byte, 256),
		}

		h.Register <- client

		// 1. Fetch existing data for this room from Postgres
		existingData, err := h.Storage.GetContent(roomID)
		if err != nil {
			log.Printf("Could not fetch initial data for room %s: %v", roomID, err)
		} else if existingData != "" {
			// 2. Send the current state immediately to the new client
			err = conn.WriteMessage(websocket.TextMessage, []byte(existingData))
			if err != nil {
				log.Printf("Initial push failed: %v", err)
			}
		}

		// Start background routines for this specific connection
		go client.WritePump()
		go client.ReadPump()
	}
}
