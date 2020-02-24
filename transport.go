package aside

import (
	"fmt"

	rs "github.com/matryer/resync"

	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	tpt "github.com/libp2p/go-libp2p-core/transport"
)

var _ tpt.Transport = (*webRTCAsideTransport)(nil)

var ProtoID = protocol.ID("/libp2p/aside/0.0.1")

var emptyWebRTCAsideMaddr, _ = ma.NewMultiaddr("/webrtc-aside")

type webRTCAsideTransport struct {
	// The host is used for the aside protocol.
	h host.Host

	// doListen avoid multiple listen to be done.
	doListen rs.Once
}

func NewWebRTCAsideTransport(h host.Host) tpt.Transport {
	return &webRTCAsideTransport{h: h}
}

func (_ *webRTCAsideTransport) CanDial(addr ma.Multiaddr) bool {
	return mafmt.WebRTCAside.Matches(addr)
}

var ErrNotWebRTCAside = fmt.Errorf("Not WebRTCAside addr.")
var ErrAlreadyListening = fmt.Errorf("There is already a listener listening.")

func (t *webRTCAsideTransport) Listen(laddr ma.Multiaddr) (tpt.Listener, error) {
	if !t.CanDial(laddr) {
		return nil, ErrNotWebRTCAside
	}

	// rL is just to export the result of Do.
	var rL *webRTCAsideListener
	// Avoid killing an other listener.
	t.doListen.Do(func() {
		rL = &webRTCAsideListener{
			t: t,
			// This channel is used by Accept to get the connection produced by the handler.
			connChan: make(chan *webRTCAsideConn),
			// This channel propagate the Close to all the goroutine.
			close: make(chan struct{}),
		}
		t.h.SetStreamHandler(ProtoID, rL.handleIncoming)
	})
	if rL == nil {
		return nil, ErrAlreadyListening
	}
	return rL, nil
}

func (_ *webRTCAsideTransport) Protocols() []int {
	return []int{ma.P_WEBRTC_ASIDE}
}

func (_ *webRTCAsideTransport) Proxy() bool {
	return false
}
