import re

with open("apps/backend-go/ws/connection.go", "r") as f:
    content = f.read()

pattern = re.compile(r'type gorillaConn struct \{\n\tid\s+string\n\tident\s+auth\.Claims\n\tconn\s+\*websocket\.Conn\n\tjoinedAt time\.Time\n\tmu\s+sync\.Mutex\n\}')
replacement = r"""type gorillaConn struct {
	id       string
	ident    auth.Claims
	conn     *websocket.Conn
	joinedAt time.Time
	mu       sync.Mutex
	send     chan []byte
	stop     chan struct{}
}

func (c *gorillaConn) writePump() {
	ticker := time.NewTicker(30 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case <-c.stop:
			return
		case payload, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
			n := len(c.send)
			for i := 0; i < n; i++ {
				c.conn.WriteMessage(websocket.TextMessage, <-c.send)
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
"""
content = pattern.sub(replacement, content)

# Remove the synchronous WriteMessage inside Send and instead push to channel
pattern2 = re.compile(r'func \(c \*gorillaConn\) Send\(ctx context\.Context, payload \[\]byte\) error \{\n\tc\.mu\.Lock\(\)\n\tdefer c\.mu\.Unlock\(\)\n\tdeadline\, ok := ctx\.Deadline\(\)\n\tif !ok \{\n\t\tdeadline = time\.Now\(\)\.Add\(5 \* time\.Second\)\n\t\}\n\t_ = c\.conn\.SetWriteDeadline\(deadline\)\n\treturn c\.conn\.WriteMessage\(websocket\.TextMessage, payload\)\n\}')
replacement2 = r"""func (c *gorillaConn) Send(ctx context.Context, payload []byte) error {
	select {
	case c.send <- payload:
		return nil
	case <-c.stop:
		return fmt.Errorf("connection closed")
	default:
		// Queue full, drop slow consumer
		c.close()
		return fmt.Errorf("slow consumer dropped")
	}
}"""
content = pattern2.sub(replacement2, content)

pattern3 = re.compile(r'func \(c \*gorillaConn\) close\(\) \{\n\tc\.mu\.Lock\(\)\n\tdefer c\.mu\.Unlock\(\)\n\t_ = c\.conn\.Close\(\)\n\}')
replacement3 = r"""func (c *gorillaConn) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.stop:
	default:
		close(c.stop)
	}
	_ = c.conn.Close()
}"""
content = pattern3.sub(replacement3, content)

with open("apps/backend-go/ws/connection.go", "w") as f:
    f.write(content)
