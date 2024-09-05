package p2p

// Peer represents a remote node in the network
type Peer interface {
}

// Transport is an interface for the
// communication between nodes in the network
type Transport interface {
	ListenAndAccept() error
}
