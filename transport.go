package aside

import (
	"fmt"

	rs "github.com/matryer/resync"

	ma "github.com/multiformats/go-multiaddr"
	mafmt "github.com/multiformats/go-multiaddr-fmt"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/protocol"
	tpt "github.com/libp2p/go-libp2p-core/transport"
	routed "github.com/libp2p/go-libp2p/p2p/host/routed"
)

var _ tpt.Transport = (*webRTCAsideTransport)(nil)

var ProtoID = protocol.ID("/libp2p/aside/0.0.1")

var emptyWebRTCAsideMaddr, _ = ma.NewMultiaddr("/webrtc-aside")

type webRTCAsideTransport struct {
	// The host is used for the aside protocol.
	h host.Host
	// We use routing by our self because we want to eject webrtc aside for the aside conn.
	r routed.Routing

	// doListen avoid multiple listen to be done.
	doListen rs.Once
}

func New(h host.Host, r routed.Routing) tpt.Transport {
	return &webRTCAsideTransport{h: h, r: r}
}

func AddTransport(h host.Host, r routed.Routing) error {
	n, ok := h.Network().(tpt.TransportNetwork)
	if !ok {
		return fmt.Errorf("%v is not a transport network", h.Network())
	}

	// TODO: Remove the transport if listen failed
	err := n.AddTransport(New(h, r))
	if err != nil {
		return err
	}
	err = n.Listen(emptyWebRTCAsideMaddr)
	if err != nil {
		return err
	}
	return nil
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
