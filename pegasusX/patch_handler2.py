import re

with open("apps/backend-go/ws/handler.go", "r") as f:
    content = f.read()

pattern = re.compile(r'gConn := &gorillaConn\{\n\t\t\tid:\s+uuid\.New\(\)\.String\(\),\n\t\t\tident:\s+ident,\n\t\t\tconn:\s+conn,\n\t\t\tjoinedAt:\s+time\.Now\(\),\n\t\t\}')
replacement = r"""gConn := &gorillaConn{
			id:       uuid.New().String(),
			ident:    ident,
			conn:     conn,
			joinedAt: time.Now(),
			send:     make(chan []byte, 256),
			stop:     make(chan struct{}),
		}
		go gConn.writePump()"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/ws/handler.go", "w") as f:
    f.write(content)
