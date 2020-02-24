package aside

import (
	"fmt"
	"net"
	"sync"

	ma "github.com/multiformats/go-multiaddr"

	tpt "github.com/libp2p/go-libp2p-core/transport"
)

type webRTCAsideListener struct {
	// The transport releated to it.
	t *webRTCAsideTransport
	// acceptChan is a channel used to get the connection created by the handler to Accept().
	connChan chan *webRTCAsideConn
	// close is a channel used to know when to close.
	close   chan struct{}
	closeMU sync.Mutex
	closed  bool
}

// Accept must not be called concurrently from multiple
func (l *webRTCAsideListener) Accept() (tpt.CapableConn, error) {
	select {
	case c := <-l.connChan:
		fmt.Println("Accepted !")
		return c, nil
	case <-l.close:
		return nil, ErrListenerAlreadyClosed
	}
}

var ErrListenerAlreadyClosed = fmt.Errorf("The listener is already closed.")

func (l *webRTCAsideListener) Close() error {
	l.closeMU.Lock()
	defer l.closeMU.Unlock()
	if !l.closed {
		close(l.close)
		l.t.h.RemoveStreamHandler(ProtoID)
		l.t.doListen.Reset()
		return nil
	}
	return ErrListenerAlreadyClosed
}

func (l *webRTCAsideListener) Addr() net.Addr {
	return emptyZeroAddr
}

var emptyZeroAddr = zeroAddr{}

// WebRTC listen on all addr, on a zero port.
type zeroAddr struct{}

func (_ zeroAddr) Network() string {
	return "udp"
}

func (_ zeroAddr) String() string {
	return ":0"
}

func (l *webRTCAsideListener) Multiaddr() ma.Multiaddr {
	// In fact we listen on libp2p but advertising listening on the p2p peer
	// listening on us lead to strange recursion issue.
	return emptyWebRTCAsideMaddr
}
