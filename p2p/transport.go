package p2p

import "net"

// Peer represents a remote node in the network
type Peer interface {
	Send([]byte) error
	RemoteAddr() net.Addr
	Close() error
}

// Transport is an interface for the
// communication between nodes in the network
type Transport interface {
	Dial(string) error
	ListenAndAccept() error

	// consume returns a recieve only channel
	Consume() <-chan RPC

	Close() error
}
