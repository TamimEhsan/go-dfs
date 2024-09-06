package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents a remote node
// in the TCP network
type TCPPeer struct {
	conn net.Conn

	// if we dial a connection => outbound true
	// if we accept a connection => inbound => outbound false
	outbound bool
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandShakeFunc HandShakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOpts
	listener net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
	}
}

// init a listener and start accept loop for
// incoming connections
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}
	fmt.Println("listening on: ", t.ListenAddr)
	go t.startAcceptLoop()
	return nil
}

// loop to accept incoming connections
func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Println("accept error: ", err)
		}
		fmt.Println("new connection from: ", conn.RemoteAddr())
		go t.handleConn(conn)
	}
}

// handle incoming connection
func (t *TCPTransport) handleConn(conn net.Conn) {

	defer func() {
		fmt.Println("closing connection: ", conn.RemoteAddr())
		conn.Close()
	}()

	peer := NewTCPPeer(conn, true)

	if err := t.HandShakeFunc(peer); err != nil {
		return
	}

	fmt.Println("Handshake completed")

	// buf := make([]byte, 1024)
	for {

		msg := Message{}
		if err := t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Println("decode error: ", err)
			return
		}
		fmt.Println("received message: ", string(msg.Payload))
	}

}
