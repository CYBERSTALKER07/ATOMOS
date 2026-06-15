package ws

// Reapable connections can be force-closed when the hub sheds stale sockets.
type Reapable interface {
	Connection
	Reap()
}
