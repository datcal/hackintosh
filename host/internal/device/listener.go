package device

import "net"

// Small indirection over net.Listen so simulator.go can stay focused on HTTP
// concerns. The interface lets tests substitute a fake listener if needed.
type lnNetListener interface {
	net.Listener
}

func netListen(network, addr string) (net.Listener, error) {
	return net.Listen(network, addr)
}
