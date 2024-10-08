package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents a remote node
// in the TCP network
type TCPPeer struct {
	net.Conn
	// if we dial a connection => outbound true
	// if we accept a connection => inbound => outbound false
	outbound bool
	//
	Wg *sync.WaitGroup // can't it be a lock?
}

type TCPTransportOpts struct {
	ListenAddr    string
	HandShakeFunc HandShakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOpts
	listener       net.Listener
	rpcCh          chan RPC
	PushAddPeer    func(Peer) error
	PushRemovePeer func(Peer) error
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		Wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(data []byte) error {
	_, err := p.Conn.Write(data)
	return err
}

func (p *TCPPeer) RemoteAddr() net.Addr {
	return p.Conn.RemoteAddr()
}

func (p *TCPPeer) CloseStream() {
	p.Wg.Done()
}

func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcCh:            make(chan RPC),
	}
}

func (t *TCPTransport) LocalAddr() string {
	return t.TCPTransportOpts.ListenAddr
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
	peer := NewTCPPeer(conn, outbound)

	defer func() {
		fmt.Println("closing connection: ", conn.RemoteAddr())
		conn.Close()
		if t.PushRemovePeer != nil {
			t.PushRemovePeer(peer)
		}
	}()

	if err := t.HandShakeFunc(peer); err != nil {
		return
	}

	if t.PushAddPeer != nil {
		if err := t.PushAddPeer(peer); err != nil {
			return
		}
	}

	fmt.Println("Handshake completed")

	for {
		msg := RPC{}
		// read the first incoming message
		// which is the metadata
		if err := t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Println("decode error: ", err)
			return
		}
		msg.From = conn.RemoteAddr().String()
		// then wait till the consumer consumes the
		// file contents and closes the stream
		peer.Wg.Add(1)
		fmt.Println("Waiting for stream from: ", conn.RemoteAddr())
		t.rpcCh <- msg

		peer.Wg.Wait()
		fmt.Println("Stream done for: ", conn.RemoteAddr())
	}

}
