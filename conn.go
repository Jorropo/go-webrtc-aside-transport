package aside

// Store generic things to conn

import (
	"fmt"
	"sync"

	ma "github.com/multiformats/go-multiaddr"

	ct "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	tpt "github.com/libp2p/go-libp2p-core/transport"

	"github.com/pion/webrtc/v2"
)

type webRTCAsideConn struct {
	t *webRTCAsideTransport

	pc *webrtc.PeerConnection

	connChan chan *webRTCAsideStream
	close    chan struct{}
	closeMU  sync.Mutex
	closed   bool

	localPeer       peer.ID
	localPrivateKey ct.PrivKey
	remotePeer      peer.ID
	remotePublicKey ct.PubKey
}

func (t *webRTCAsideTransport) upgradeConn(pc *webrtc.PeerConnection, s network.Stream) *webRTCAsideConn {
	sC := s.Conn()
	c := &webRTCAsideConn{
		t:               t,
		pc:              pc,
		connChan:        make(chan *webRTCAsideStream),
		close:           make(chan struct{}),
		localPeer:       sC.LocalPeer(),
		localPrivateKey: sC.LocalPrivateKey(),
		remotePeer:      sC.RemotePeer(),
		remotePublicKey: sC.RemotePublicKey(),
	}
	pc.OnDataChannel(c.handleDataChannel)
	pc.OnConnectionStateChange(c.handleConnectionStatesChange)
	return c
}

func (c *webRTCAsideConn) handleDataChannel(d *webrtc.DataChannel) {
	d.OnOpen(func() {
		s, err := upgradeDataChannel(d)
		if err != nil {
			return
		}
		select {
		case c.connChan <- s:
		case <-c.close:
			s.Close()
		}
	})
}

func (c *webRTCAsideConn) handleConnectionStatesChange(state webrtc.PeerConnectionState) {
	if state == webrtc.PeerConnectionStateClosed || state == webrtc.PeerConnectionStateDisconnected {
		c.Close()
	}
}

func (c *webRTCAsideConn) Transport() tpt.Transport {
	return c.t
}

var ErrConnAlreadyClosed = fmt.Errorf("The connection is already closed.")

// Muxing stuff
func (c *webRTCAsideConn) Close() error {
	c.closeMU.Lock()
	defer c.closeMU.Unlock()
	if !c.closed {
		close(c.close)
		c.pc.Close()
		close(c.connChan)
		c.closed = true
		return nil
	}
	return ErrConnAlreadyClosed
}

func (c *webRTCAsideConn) IsClosed() bool {
	c.closeMU.Lock()
	defer c.closeMU.Unlock()
	return c.closed
}

func (c *webRTCAsideConn) OpenStream() (mux.MuxedStream, error) {
	d, err := c.pc.CreateDataChannel("", nil)
	if err != nil {
		return nil, err
	}
	s, err := upgradeDataChannel(d)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (c *webRTCAsideConn) AcceptStream() (mux.MuxedStream, error) {
	select {
	case s := <-c.connChan:
		return s, nil
	case <-c.close:
		return nil, ErrConnAlreadyClosed
	}
}

// Maddr stuff
func (_ *webRTCAsideConn) LocalMultiaddr() ma.Multiaddr {
	return emptyWebRTCAsideMaddr
}

func (_ *webRTCAsideConn) RemoteMultiaddr() ma.Multiaddr {
	return emptyWebRTCAsideMaddr
}

// Security stuff
// All the security is a crypto chain, newer webrtc keys are assumed safe
// because they are exchanged through a safe connection.
func (c *webRTCAsideConn) LocalPeer() peer.ID {
	return c.localPeer
}

func (c *webRTCAsideConn) LocalPrivateKey() ct.PrivKey {
	return c.localPrivateKey
}

func (c *webRTCAsideConn) RemotePeer() peer.ID {
	return c.remotePeer
}

func (c *webRTCAsideConn) RemotePublicKey() ct.PubKey {
	return c.remotePublicKey
}
