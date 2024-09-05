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

type TCPTransport struct {
	listenAddress string
	listener      net.Listener

	mu    sync.RWMutex
	peers map[net.Addr]Peer
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func NewTCPTransport(listenAddress string) *TCPTransport {
	return &TCPTransport{
		listenAddress: listenAddress,
		peers:         make(map[net.Addr]Peer),
	}
}

// init a listener and start accept loop for
// incoming connections
func (t *TCPTransport) ListenAndAccept() error {
	var err error
	t.listener, err = net.Listen("tcp", t.listenAddress)
	if err != nil {
		return err
	}
	fmt.Println("listening on: ", t.listenAddress)
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

	peer := NewTCPPeer(conn, true)
	fmt.Println("new connection from: ", peer)
	
}
