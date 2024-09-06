package p2p

// Message holds any arbitrary data
// that can be sent over the network
type RPC struct {
	From    string
	Payload []byte
}
