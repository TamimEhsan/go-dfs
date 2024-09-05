package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTCPTransport(t *testing.T) {
	listenAddress := ":4000"
	tr := NewTCPTransport(listenAddress)
	assert.Equal(t, listenAddress, tr.listenAddress)
	
	// server 
	tr.ListenAndAccept()

	// t.listener.Accept()
}
