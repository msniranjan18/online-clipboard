package hub

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 50 // 50KB limit for text
)

type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	RoomID string
	Send   chan []byte
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, bytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		// Try to parse the JSON. If it fails, treat it as a normal update.
		if err := json.Unmarshal(bytes, &msg); err != nil {
			msg = Message{
				RoomID:  c.RoomID,
				Content: string(bytes),
				Action:  "UPDATE",
			}
		}

		// Ensure RoomID is set if the frontend forgot it
		if msg.RoomID == "" {
			msg.RoomID = c.RoomID
		}

		// 1. Broadcast to local clients
		c.Hub.Broadcast <- msg

		// 2. Publish to Redis for other server instances
		payload, _ := json.Marshal(msg)
		c.Hub.Storage.RDB.Publish(c.Hub.Storage.Ctx, "clipboard_sync", payload)

		// 3. Persist to DB (Ideally debounced)
		go c.Hub.Storage.SaveContent(msg.RoomID, msg.Content)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
