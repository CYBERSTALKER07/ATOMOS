package ws

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// PingInterval is how often we send pings to clients.
	PingInterval = 15 * time.Second

	// PongWait is the maximum time we wait for a pong response.
	// If 2 consecutive pings go unanswered, the connection is dead.
	PongWait = 30 * time.Second

	// WriteWait is the time allowed to write a message to the peer.
	WriteWait = 10 * time.Second
)

// ConfigureKeepalive sets up ping/pong handling for a WebSocket connection.
// It sets the read deadline and pong handler, and starts a background goroutine
// that sends pings every PingInterval. The goroutine stops when done is closed.
//
// Usage:
//
//	done := ws.ConfigureKeepalive(conn)
//	defer close(done)
func ConfigureKeepalive(conn *websocket.Conn) chan struct{} {
	done := make(chan struct{})

	// Set initial read deadline — extended by each pong
	conn.SetReadDeadline(time.Now().Add(PongWait))
	conn.SetPongHandler(func(appData string) error {
		conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	// Background ping sender
	go func() {
		ticker := time.NewTicker(PingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(WriteWait))
				if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(WriteWait)); err != nil {
					log.Printf("[WS_KEEPALIVE] Ping failed — connection likely dead: %v", err)
					return
				}
			}
		}
	}()

	return done
}
