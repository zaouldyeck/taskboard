package websocket

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to peer.
	writeWait = 10 * time.Second

	// Time allowed to read next pong message from peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period.
	pingPeriod = (pongWait * 9) / 10

	// Maximum allowed message size from peer.
	maxMessageSize = 512
)

// Client represents a single websocket connection.
type Client struct {
	Hub  *Hub
	Conn *websocket.Conn

	// Buffered channel for outbound messages.
	Send chan []byte
}

// ReadMsgFromWebSocket reads messages from websocket and sends them to hub.
func (c *Client) ReadMsgFromWebSocket() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure) {
				log.Printf("Websocket error: %v", err)
			}
			break
		}

		// For now, just log client messages.
		log.Printf("Received from client: %s", message)
	}
}

// WriteMsgToWebSocket reads message from hub and sends them to the websocket.
func (c *Client) WriteMsgToWebSocket() {
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
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
