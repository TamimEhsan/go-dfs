package p2p

import (
	"fmt"
	"net"
)

// TCPPeer represents a remote node
// in the TCP network
type TCPPeer struct {
	net.Conn
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
	rpcCh    chan RPC
	PushPeer func(Peer) error
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Send(data []byte) error {
	_, err := p.Conn.Write(data)
	return err
}

func (p *TCPPeer) RemoteAddr() net.Addr {
	return p.Conn.RemoteAddr()
}

func (p *TCPPeer) Close() error {
	return p.Conn.Close()
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcCh:            make(chan RPC),
	}
}

// consume implements the transport interface
// and returns a receive only channel to consume
// incoming messages
func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcCh
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Dial(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	go t.handleConn(conn, true)
	return nil
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

		if err == net.ErrClosed {
			fmt.Println("listener closed")
			return
		}

		if err != nil {
			fmt.Println("accept error: ", err)
		}
		fmt.Println("new connection from: ", conn.RemoteAddr())
		go t.handleConn(conn, false)
	}

}

// handle incoming connection
func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {

	defer func() {
		fmt.Println("closing connection: ", conn.RemoteAddr())
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)

	if err := t.HandShakeFunc(peer); err != nil {
		return
	}

	if t.PushPeer != nil {
		if err := t.PushPeer(peer); err != nil {
			return
		}
	}

	fmt.Println("Handshake completed")

	for {
		msg := RPC{}
		if err := t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Println("decode error: ", err)
			return
		}
		
		msg.From = conn.RemoteAddr().String()
		t.rpcCh <- msg
	}

}
