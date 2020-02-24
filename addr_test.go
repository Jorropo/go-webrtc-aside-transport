package aside

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
)

func TestCanDial(t *testing.T) {
	addrAside, err := ma.NewMultiaddr("/webrtc-aside")
	if err != nil {
		t.Fatal(err)
	}
	addrTCP, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/5555")
	if err != nil {
		t.Fatal(err)
	}

	d := &WebRTCAsideTransport{}
	if !d.CanDial(addrAside) {
		t.Fatal("expected to match webrtc aside maddr, but did not")
	}
	if d.CanDial(addrTCP) {
		t.Fatal("expected to not match tcp maddr, but did")
	}
}
