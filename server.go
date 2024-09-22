package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/tamimehsan/go-distributed-fs/p2p"
)

type FileServerOpts struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	connectionLock sync.Mutex
	connections    map[string]bool

	store  *Store
	quitCh chan struct{}
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:          opts.StorageRoot,
		PathTransform: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
		connections:    make(map[string]bool),
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
	}
}

type Message struct {
	Payload any
}

type MessageStoreFile struct {
	Key  string
	Size int64
}

type MessageGetFile struct {
	Key string
}

type MessagePeerDiscovery struct {
	Req  string
	Addr string
}

type MessageSendAddr struct {
	Addr string
}

func (s *FileServer) broadcast(msg *Message, encode bool) error {
	buf := new(bytes.Buffer)
	if encode {
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			return err
		}
	} else {
		buf.Write(msg.Payload.([]byte))
	}

	for _, peer := range s.peers {

		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}

	}

	return nil
}

func (s *FileServer) multicast(msg *Message, peers []io.Writer, encode bool) error {
	buf := new(bytes.Buffer)
	if encode {
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			return err
		}
	} else {
		buf.Write(msg.Payload.([]byte))
	}

	mw := io.MultiWriter(peers...)
	_, err := io.Copy(mw, buf)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileServer) ReadData(key string) (int64, io.Reader, error) {

	if s.store.Exists(key) {
		return s.store.Read(key)
	}

	fmt.Println("File doesn't exist on server, attempting to retrieve from network")
	p := &Message{
		Payload: MessageGetFile{
			Key: key,
		},
	}
	err := s.broadcast(p, true)
	if err != nil {
		return 0, nil, err
	}

	time.Sleep(time.Second * 3)

	return s.store.Read(key)

}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	// read the data from the reader and
	fileBuffer := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuffer)

	// check if the file already exists
	exists := s.store.Exists(key)

	// store it in the file store
	if err := s.store.Write(key, tee); err != nil {
		return err
	}

	// if the file already exists, no need to gossip
	if exists {
		return nil
	}
	// gossip the file metadata to peers
	p := &Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: int64(fileBuffer.Len()),
		},
	}

	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	// randomly shuffle the peers
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	if len(peers) > 2 {
		peers = peers[:2]
	}

	err := s.multicast(p, peers, true)
	if err != nil {
		return err
	}

	// stream the file contents to peers
	time.Sleep(time.Millisecond * 5)

	mw := io.MultiWriter(peers...)
	_, err = io.Copy(mw, fileBuffer)
	if err != nil {
		return err
	}
	return nil
}

func (s *FileServer) Stop() {
	close(s.quitCh)
	fmt.Println("server signaled to stop")
}

func (s *FileServer) AddPeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[peer.RemoteAddr().String()] = peer

	s.connectionLock.Lock()
	defer s.connectionLock.Unlock()
	s.connections[peer.RemoteAddr().String()] = true
	s.connections[peer.LocalAddr().String()] = true
	fmt.Println("peer added: ", peer.RemoteAddr().String(), "+", peer.LocalAddr().String())
	return nil
}

func (s *FileServer) RemovePeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	delete(s.peers, peer.RemoteAddr().String())
	s.peerLock.Unlock()
	s.connectionLock.Lock()
	delete(s.connections, peer.RemoteAddr().String())
	delete(s.connections, peer.LocalAddr().String())
	s.connectionLock.Unlock()
	fmt.Println("peer removed: ", peer.RemoteAddr().String(), "+", peer.LocalAddr().String())
	return nil
}

func (s *FileServer) bootstrapNetwork() {
	for _, peer := range s.BootstrapNodes {
		if len(peer) == 0 {
			continue
		}
		go func(peer string) {
			if err := s.Transport.Dial(peer); err != nil {
				fmt.Println("error dialing peer: ", err)
			}
		}(peer)
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return err
	}
	s.bootstrapNetwork()
	time.Sleep(time.Second * 3)
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	s.multicast(&Message{Payload: MessageSendAddr{Addr: s.Transport.LocalAddr()}}, peers, true)

	p := &Message{
		Payload: MessagePeerDiscovery{
			Addr: s.Transport.LocalAddr(),
			Req:  "peer_discovery",
		},
	}
	err := s.multicast(p, peers, true)
	if err != nil {
		fmt.Println("error sending peer discovery message: ", err)
		return err
	}
	fmt.Println("server started on: ", s.Transport.LocalAddr())
	s.loop()

	return nil
}

func (s *FileServer) loop() {
	defer s.Transport.Close()
	for {
		select {
		case <-s.quitCh:
			return
		case rpc := <-s.Transport.Consume():

			// decode the file metadata at first from the rpc
			var p Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&p); err != nil {
				log.Println("error decoding rpc: ", err)
			}

			s.handleMessage(rpc.From, &p)

		}
	}
}

func (s *FileServer) handleMessage(from string, msg *Message) error {

	switch msg.Payload.(type) {
	case MessageStoreFile:
		return s.handleStoreFile(from, msg.Payload.(MessageStoreFile))
	case MessageGetFile:
		return s.handleGetFile(from, msg.Payload.(MessageGetFile))
	case MessagePeerDiscovery:
		return s.handlePeerDiscovery(from, msg.Payload.(MessagePeerDiscovery))
	case MessageSendAddr:
		return s.handleSendAddr(from, msg.Payload.(MessageSendAddr))
	default:

	}
	return nil
}

func (s *FileServer) handleSendAddr(from string, msg MessageSendAddr) error {
	peer, ok := s.peers[from]
	if !ok {
		log.Println("peer not found: ", from)
		return nil
	}
	defer peer.CloseStream()
	s.connectionLock.Lock()
	defer s.connectionLock.Unlock()
	s.connections[msg.Addr] = true
	return nil
}

func (s *FileServer) handlePeerDiscovery(from string, msg MessagePeerDiscovery) error {
	fmt.Println("peer discovery message received from: ", from, " addr: ", msg.Addr)
	peer, ok := s.peers[from]
	if !ok {
		log.Println("peer not found: ", from)
		return nil
	}
	defer peer.CloseStream()
	// check if the peer is already in the list
	if msg.Addr == s.Transport.LocalAddr() {
		fmt.Println("peer is the same as the local address")
		return nil
	}
	if _, ok := s.connections[msg.Addr]; !ok {
		go func(peer string) {
			if err := s.Transport.Dial(peer); err != nil {
				fmt.Println("error dialing peer: ", err)
			}
		}(msg.Addr)

		time.Sleep(time.Second * 3)
		peer = s.peers[msg.Addr]
		if peer == nil {
			fmt.Println("peer not found after dialing")
			return nil
		}
		s.multicast(&Message{Payload: MessageSendAddr{Addr: s.Transport.LocalAddr()}}, []io.Writer{peer}, true)

	} else if msg.Req == "peer_gossip" {
		return nil
	}

	// gossip
	p := &Message{
		Payload: MessagePeerDiscovery{
			Addr: msg.Addr,
			Req:  "peer_gossip",
		},
	}
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}
	// randomly shuffle the peers
	rand.Shuffle(len(peers), func(i, j int) {
		peers[i], peers[j] = peers[j], peers[i]
	})
	if len(peers) > 2 {
		peers = peers[:2]
	}

	err := s.multicast(p, peers, true)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileServer) handleStoreFile(from string, msg MessageStoreFile) error {
	key := msg.Key
	fmt.Println("file metadata received on server ", s.StorageRoot, " from ", from, " : ", key)

	peer, ok := s.peers[from]
	if !ok {
		log.Println("peer not found: ", from)
		return nil
	}
	defer peer.CloseStream()

	// read the file contents from the stream and write to file
	s.StoreData(key, io.LimitReader(peer, msg.Size))

	fmt.Println("file content received on server ", s.StorageRoot, " from ", from, " : ", key)
	// Close stream decreaments the zemaphore thus
	// the peer will end the waiting

	return nil
}

func (s *FileServer) handleGetFile(from string, msg MessageGetFile) error {
	if !s.store.Exists(msg.Key) {
		fmt.Println("file doesn't exist here too!")
		return nil
	}

	peer, ok := s.peers[from]
	if !ok {
		fmt.Println("Peer not found")
		return nil
	}
	defer peer.CloseStream()

	fmt.Println("file retrieved to server to network!")

	n, r, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}

	defer r.(io.ReadCloser).Close()

	p := &Message{
		Payload: MessageStoreFile{
			Key:  msg.Key,
			Size: int64(n),
		},
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(p); err != nil {
		return err
	}

	err = peer.Send(buf.Bytes())
	if err != nil {
		return err
	}

	// stream the file contents to the peer

	_, err = io.Copy(peer, r)
	// fmt.Println("attempting to send file data:::", fileBuffer.String())
	if err != nil {
		return err
	}

	return err
}

func init() {
	gob.Register(MessageStoreFile{})
	gob.Register(MessageGetFile{})
	gob.Register(MessagePeerDiscovery{})
	gob.Register(MessageSendAddr{})
}
