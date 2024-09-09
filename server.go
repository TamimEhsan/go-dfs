package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

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
		store:          NewStore(storeOpts),
		quitCh:         make(chan struct{}),
	}
}

type Message struct {
	Payload any
}

func (s *FileServer) broadcast(msg *Message) error {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}

	for _, peer := range s.peers {

		if err := peer.Send(buf.Bytes()); err != nil {
			return err
		}

	}

	return nil
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	buf := new(bytes.Buffer)
	tee := io.TeeReader(r, buf)

	if err := s.store.Write(key, tee); err != nil {
		return err
	}

	p := &Message{
		Payload: buf.Bytes(),
	}

	return s.broadcast(p)
}

func (s *FileServer) Stop() {
	close(s.quitCh)
	fmt.Println("server signaled to stop")
}

func (s *FileServer) AddPeer(peer p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[peer.RemoteAddr().String()] = peer
	fmt.Println("peer added: ", peer.RemoteAddr().String())
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

			var p Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&p); err != nil {
				log.Println("error decoding rpc: ", err)
			}

			fmt.Println("recieved message", string(p.Payload.([]uint8)))

			// fmt.Println("rpc received on server ", s.StorageRoot, ": ", string(rpc.Payload))

			// handle rpc
		}
	}
}

func (s *FileServer) Store(key string, r io.Reader) error {

	return nil
}
