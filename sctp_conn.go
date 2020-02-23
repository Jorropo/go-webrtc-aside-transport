package aside

import (
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/pion/webrtc/v2"
)

type webRTCAsideSCTPConn struct {
  webRTCAsideBaseConn
}

func (t *WebRTCAsideTransport) upgradeSCTPConn(pc *webrtc.PeerConnection, s network.Stream) (*webRTCAsideConn, error) {
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

// Muxing stuff
func (_ *webRTCAsideSCTPConn) Close() error {
	return nil
}

func (c *webRTCAsideSCTPConn) IsClosed() bool {
	return c.pc.ICEConnectionState() == webrtc.ICEConnectionStateClosed
}

func (_ *webRTCAsideSCTPConn) OpenStream() (mux.MuxedStream, error) {
	return nil, nil
}

func (_ *webRTCAsideSCTPConn) AcceptStream() (mux.MuxedStream, error) {
	return nil, nil
}
