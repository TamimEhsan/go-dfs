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

type MessageStoreFile struct {
	Key  string
	Size int64
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

func (s *FileServer) StoreData(key string, r io.Reader) error {
	// read the data from the reader and

	fileBuffer := new(bytes.Buffer)
	tee := io.TeeReader(r, fileBuffer)

	// store it in the file store
	if err := s.store.Write(key, tee); err != nil {
		return err
	}

	// broadcast the file metadata to peers
	p := &Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: int64(fileBuffer.Len()),
		},
	}
	err := s.broadcast(p, true)
	if err != nil {
		return err
	}

	// stream the file contents to peers
	p = &Message{
		Payload: fileBuffer.Bytes(),
	}
	// for now, just broadcast normally without encoding
	err = s.broadcast(p, false)
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
	default:

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

	// read the file contents from the stream
	/*
		fileContent := make([]byte, 1024)

		if _, err := peer.Read(fileContent); err != nil {
			log.Println("error reading file contents: ", err)
		}

		fmt.Println("file content received on server ", s.StorageRoot, ": ", string(fileContent))
	*/
	if err := s.store.Write(key, io.LimitReader(peer, msg.Size)); err != nil {
		log.Println("error writing file contents: ", err)
	}
	fmt.Println("file content received on server ", s.StorageRoot, " from ", from, " : ", key)
	// Close stream decreaments the zemaphore thus
	// the peer will end the waiting
	peer.CloseStream()
	return nil
}

func (s *FileServer) Store(key string, r io.Reader) error {

	return nil
}

func init() {
	gob.Register(MessageStoreFile{})

}
