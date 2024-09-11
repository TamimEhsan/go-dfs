package p2p

import "net"

// Peer represents a remote node in the network
type Peer interface {
	// underlying connection of the peer
	net.Conn

	Send([]byte) error

	CloseStream()
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
