package ws

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Loosen origin check for scaffold; production restricts this.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// gorillaConn wraps a gorilla/websocket connection to implement ws.Connection.
type gorillaConn struct {
	id    string
	ident auth.Claims
	conn  *websocket.Conn
}

func (c *gorillaConn) ID() string {
	return c.id
}

func (c *gorillaConn) Identity() auth.Claims {
	return c.ident
}

func (c *gorillaConn) Send(ctx context.Context, payload []byte) error {
	// Bounded write deadline to prevent slow clients from blocking the hub.
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	_ = c.conn.SetWriteDeadline(deadline)
	return c.conn.WriteMessage(websocket.TextMessage, payload)
}
