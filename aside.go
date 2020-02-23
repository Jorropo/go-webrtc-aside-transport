package aside

import (
	"fmt"

	pb "github.com/Jorropo/go-webrtc-aside-transport/pb"
	proto "github.com/gogo/protobuf/proto"

	"github.com/libp2p/go-libp2p-core/network"
)

var peerConnectionConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			// TODO: Find stun server in the DHT.
			// TODO: Stop using google stun, maybe a PL or mozilla one ?
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

// Ordered by preference, lower index is better, don't add something here without implementing it.
var supportedProtos = []pb.Proto{
	pb.Proto_DATA_CHANNEL_TRICKLE,
	pb.Proto_DATA_CHANNEL,
}

var webrtcAPI *webrtc.API

func init() {
	s := webrtc.SettingEngine{}
	s.SetTrickle(true)
	s.DetachDataChannels()
	webrtcAPI = webrtc.NewAPI(webrtc.WithSettingEngine(s))
}

func (l *webRTCAsideListener) handleIncoming(s network.Stream) {
	defer s.Close()

	// TODO: Support STUN Share.
	// Sending our hello.
	b, err := proto.Marshal(&pb.Hello{Protos: supportedProtos})
	if err != nil {
		// TODO: Log somewhere, also for all error return of this function.
		return
	}
	_, err = s.Write(b)
	if err != nil {
		return
	}
	rb := []byte{}
	n, err := s.Read(rb)
	if err != nil {
		return
	}

	// Parsing remote Hello
	rHello := &pb.Hello{}
	proto.Unmarshal(rb[:n], rHello)
	// Does it require something we can't do ?
	if rHello.GetNeedStun() || rHello.GetProtos() == nil {
		// TODO: Send a Terminate message with the reason, also for other returns.
		return
	}
	// Does we have a proto in common ?
	negociated, err = negociateProto(rHello.GetProtos(), supportedProtos)
	if negociated == pb.NO_PROTO {
		return
	}
	// Send an empty STUN Server list if remote asked.
	if rHello.GetWantStun() || rHello.GetWantGoodStun() {
		b, err := proto.Marshal(&pb.StunShare{StunServers: []*pb.StunServer{}})
		if err != nil {
			return
		}
		_, err = s.Write(b)
		if err != nil {
			return
		}
	}

	var (
		trickle = false
		c tpt.CapableConn
	)
	switch negociated {
	case pb.Proto_SCTP_TRICKLE:
		trickle = true
		fallthrough
	case pb.Proto_SCTP:
		gatherer, err := webrtcAPI.NewICEGatherer(config)
		if err != nil {
			return
		}
		ice := webrtcAPI.NewICETransport(gatherer)
		dtls, err := webrtcAPI.NewDTLSTransport(ice, nil)
		if err != nil {
			return
		}
		sctp := webrtcAPI.NewSCTPTransport(dtls)
		c, err =
	}
	select {
	// Trying to return to accept.
	case l.connChan <- c:
		// If the listener is closed we should die silently.
	case <-l.close:
		c.Close()
	}
}

func negociateProto(incoming, outgoing []pb.Proto) pb.Proto {
	for _, i := range incoming {
		for _, j := range outgoing {
			if i == j {
				return i
			}
		}
	}
	return pb.NO_PROTO
}
