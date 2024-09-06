package p2p

type HandShakeFunc func(Peer) error

func NOPHandshake(Peer) error {
	return nil
}
