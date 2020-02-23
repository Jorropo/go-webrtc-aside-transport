package aside

import (
	"context"
	"fmt"

	rs "github.com/matryer/resync"

	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	tpt "github.com/libp2p/go-libp2p-core/transport"
	"github.com/libp2p/go-libp2p/p2p/host/routed"
)

var _ tpt.Transport = (*WebRTCAsideTransport)(nil)

var ProtoID = protocol.ID("/libp2p/aside/0.0.1")

var emptyWebRTCAsideMaddr, _ = ma.NewMultiaddr("/webrtc-aside")

type WebRTCAsideTransport struct {
	// The host is used for the aside protocol.
	h *routedhost.RoutedHost

	// doListen avoid multiple listen to be done.
	doListen rs.Once
}

func (t *WebRTCAsideTransport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (tpt.CapableConn, error) {
	if !t.CanDial(raddr) {
		return nil, ErrNotWebRTCAside
	}
	_, err := t.h.NewStream(ctx, p, ProtoID)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (_ *WebRTCAsideTransport) CanDial(addr ma.Multiaddr) bool {
	return mafmt.WebRTCAside.Matches(addr)
}

var ErrNotWebRTCAside = fmt.Errorf("Not WebRTCAside addr.")
var ErrAlreadyListening = fmt.Errorf("There is already a listener listening.")

func (t *WebRTCAsideTransport) Listen(laddr ma.Multiaddr) (tpt.Listener, error) {
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
			connChan: make(chan tpt.CapableConn),
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

func (_ *WebRTCAsideTransport) Protocols() []int {
	return []int{ma.P_WEBRTC_ASIDE}
}

func (_ *WebRTCAsideTransport) Proxy() bool {
	return false
}
