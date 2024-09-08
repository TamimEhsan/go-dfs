package p2p

// Peer represents a remote node in the network
type Peer interface {
	Close() error
}

// Transport is an interface for the
// communication between nodes in the network
type Transport interface {
	ListenAndAccept() error

	// consume returns a recieve only channel 
	Consume() <-chan RPC

	Close() error
}
