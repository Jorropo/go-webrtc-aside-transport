package aside
// Store generic things to conn

import (
	ma "github.com/multiformats/go-multiaddr"

	ct "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	tpt "github.com/libp2p/go-libp2p-core/transport"

	"github.com/pion/webrtc/v2"
)

type webRTCAsideBaseConn struct {
	t *WebRTCAsideTransport

	localPeer       peer.ID
	localPrivateKey ct.PrivKey
	remotePeer      peer.ID
	remotePublicKey ct.PubKey
}

func (t *WebRTCAsideTransport) upgradeDataChannelConn(pc *webrtc.PeerConnection, s network.Stream) (*webRTCAsideConn, error) {
	pc.OnDataChannel()
	c := s.Conn()
	return &webRTCAsideConn{
		t:               t,
		pc:              pc,
		localPeer:       c.LocalPeer(),
		localPrivateKey: c.LocalPrivateKey(),
		remotePeer:      c.RemotePeer(),
		remotePublicKey: c.RemotePublicKey(),
	}, nil
}

func (c *webRTCAsideBaseConn) handleDataChannel(d *webrtc.DataChannel) tpt.Transport {
	c.connChan <- d.Detach
}

func (c *webRTCAsideBaseConn) Transport() tpt.Transport {
	return c.t
}

// Maddr stuff
func (_ *webRTCAsideBaseConn) LocalMultiaddr() ma.Multiaddr {
	return emptyWebRTCAsideMaddr
}

func (_ *webRTCAsideBaseConn) RemoteMultiaddr() ma.Multiaddr {
	return emptyWebRTCAsideMaddr
}

// Security stuff
// All the security is a crypto chain, newer webrtc keys are assumed safe
// because they are exchanged through a safe connection.
func (c *webRTCAsideBaseConn) LocalPeer() peer.ID {
	return c.localPeer
}

func (c *webRTCAsideBaseConn) LocalPrivateKey() ct.PrivKey {
	return c.localPrivateKey
}

func (c *webRTCAsideBaseConn) RemotePeer() peer.ID {
	return c.remotePeer
}

func (c *webRTCAsideBaseConn) RemotePublicKey() ct.PubKey {
	return c.remotePublicKey
}
