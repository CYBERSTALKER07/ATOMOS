import re

with open("apps/backend-go/ws/handler.go", "r") as f:
    content = f.read()

pattern = re.compile(r'gConn := &gorillaConn\{\n\t\t\tid:\s+connID,\n\t\t\tident:\s+ident,\n\t\t\tconn:\s+c,\n\t\t\tjoinedAt:\s+time\.Now\(\),\n\t\t\}')
replacement = r"""gConn := &gorillaConn{
			id:       connID,
			ident:    ident,
			conn:     c,
			joinedAt: time.Now(),
			send:     make(chan []byte, 256),
			stop:     make(chan struct{}),
		}
		go gConn.writePump()"""
content = pattern.sub(replacement, content)

# Remove ping loop from handler.go since writePump does pinging now
pattern2 = re.compile(r'startPingLoop\(conn, done, log\)')
replacement2 = r"""// startPingLoop(conn, done, log) // handled by writePump now"""
content = pattern2.sub(replacement2, content)

with open("apps/backend-go/ws/handler.go", "w") as f:
    f.write(content)
