package aside

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/Jorropo/go-webrtc-aside-transport/pb"
	proto "github.com/gogo/protobuf/proto"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	tpt "github.com/libp2p/go-libp2p-core/transport"

	"github.com/pion/webrtc/v2"
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

const connectionWaitTime time.Duration = 35 * time.Second

// A time to sleep after the first ice is working before terminating.
const iceGraceTime time.Duration = 25 * time.Second

var webrtcAPI *webrtc.API

func init() {
	s := webrtc.SettingEngine{}
	s.SetTrickle(true)
	s.DetachDataChannels()
	webrtcAPI = webrtc.NewAPI(webrtc.WithSettingEngine(s))
}

func (l *webRTCAsideListener) handleIncoming(s network.Stream) {
	// ICE locking is here to make the event goroutine waiting the right moment before firing write.
	iceLocked := true
	var icewg sync.WaitGroup
	icewg.Add(1)
	defer func() {
		if iceLocked {
			icewg.Done()
		}
	}()
	defer s.Close()
	endingBad := true

	// Creating the PeerConnection object
	pc, err := webrtcAPI.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		// TODO: Log somewhere, also for all error return of this function.
		return
	}
	defer func() {
		// If we end its better to close to avoid memory leaks, but if evrythings worked well no need to close this.
		if endingBad {
			pc.Close()
		}
	}()
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		// Copying for protobuf
		cand := c.ToJSON().Candidate
		b, err := proto.Marshal(&pb.Candidate{Ice: &cand})
		if err != nil {
			return
		}
		// Waiting for ice time.
		icewg.Wait()
		s.Write(b)
	})

	// TODO: Support STUN Share.
	// Sending our hello.
	b, err := proto.Marshal(&pb.Hello{})
	if err != nil {
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
	{
		rHello := &pb.Hello{}
		proto.Unmarshal(rb[:n], rHello)
		// Does it require something we can't do ?
		if rHello.GetNeedStun() {
			// TODO: Send a Terminate message with the reason, also for other returns.
			return
		}
		// Send an empty STUN Server list if remote asked.
		if rHello.GetWantStun() || rHello.GetWantGoodStun() {
			b, err = proto.Marshal(&pb.StunShare{StunServers: []*pb.StunServer{}})
			if err != nil {
				return
			}
			_, err = s.Write(b)
			if err != nil {
				return
			}
		}
	}

	// Setting the SDP
	{
		n, err = s.Read(rb)
		if err != nil {
			return
		}
		rSdp := &pb.SDP{}
		proto.Unmarshal(rb[:n], rSdp)
		err = pc.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  rSdp.GetSdp(),
		})
		if err != nil {
			return
		}
		answer, err := pc.CreateAnswer(nil)
		if err != nil {
			return
		}
		err = pc.SetLocalDescription(answer)
		if err != nil {
			return
		}
		b, err = proto.Marshal(&pb.SDP{Sdp: &answer.SDP})
		if err != nil {
			return
		}
		_, err = s.Write(b)
		if err != nil {
			return
		}
	}

	// Getting ICE candidates
	{
		icewg.Done()
		iceLocked = false
		// Checking if a connection worked.
		timeoutChan := make(chan struct{})
		successChan := make(chan struct{})
		pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			if state == webrtc.PeerConnectionStateConnected {
				select {
				case <-timeoutChan:
				case successChan <- struct{}{}:
				}
			}
		})
		var iceCount uint64 = 0
		for {
			n, err = s.Read(rb)
			if err != nil {
				return
			}
			// Parsing the message.
			rWSDP := &pb.WrapExchange{}
			proto.Unmarshal(rb[:n], rWSDP)
			// Checking message type.
			switch v := rWSDP.GetM().(type) {
			case *pb.WrapExchange_Terminate:
				return
			case *pb.WrapExchange_Ice:
				iceCount++
				// Adding the ICE candidate
				err = pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: v.Ice.GetIce()})
				if err != nil {
					return
				}
			case *pb.WrapExchange_Genend:
				if iceCount > 0 {
					// Sending the pong genend message.
					b, err = proto.Marshal(
						&pb.WrapExchange{
							M: &pb.WrapExchange_Genend{
								Genend: &pb.GenerationEnded{},
							},
						},
					)
					if err != nil {
						return
					}
					_, err = s.Write(b)
					if err != nil {
						return
					}
					break
				}
				// Terminating if no ICE candidate was received.
				return
			}
		}
		// Checking connection state.
		go func() {
			time.Sleep(connectionWaitTime)
			close(timeoutChan)
		}()
		select {
		case <-successChan:
		case <-timeoutChan:
			// TODO: Implement retry
			return
		}
	}

	// Finishing
	c := l.t.upgradeConn(pc, s)
	// Setting endingBad to true because if closing `c.Close()` is gonna take care of closing `pc`.
	endingBad = false
	select {
	case l.connChan <- c: // Trying to return to accept.
		time.Sleep(iceGraceTime)
		// Needing to copy it because `pb.Reason_SUCCESS` is a const.
		reason := pb.Reason_SUCCESS
		b, err = proto.Marshal(
			&pb.WrapExchange{
				M: &pb.WrapExchange_Terminate{
					Terminate: &pb.Terminate{
						Reason: &reason,
					},
				},
			},
		)
		if err != nil {
			return
		}
		_, err = s.Write(b)
		if err != nil {
			return
		}
	case <-l.close: // If the listener is closed we should die silently.
		c.Close()
	}
}

var ErrTimeout = fmt.Errorf("Connection Timeout.")

func (t *webRTCAsideTransport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (tpt.CapableConn, error) {
	if !t.CanDial(raddr) {
		return nil, ErrNotWebRTCAside
	}
	s, err := t.h.NewStream(ctx, p, ProtoID)
	if err != nil {
		return nil, err
	}
	// ICE locking is here to make the event goroutine waiting the right moment before firing write.
	iceLocked := true
	var icewg sync.WaitGroup
	icewg.Add(1)
	defer func() {
		if iceLocked {
			icewg.Done()
		}
	}()
	endingBad := true
	defer s.Close()

	// Creating the PeerConnection object
	pc, err := webrtcAPI.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		return nil, err
	}
	defer func() {
		// If we end its better to close to avoid memory leaks, but if evrythings worked well no need to close this.
		if endingBad {
			pc.Close()
		}
	}()
	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		// Copying for protobuf
		cand := c.ToJSON().Candidate
		b, err := proto.Marshal(&pb.Candidate{Ice: &cand})
		if err != nil {
			return
		}
		// Waiting for ice time.
		icewg.Wait()
		s.Write(b)
	})

	// Sending our hello.
	b, err := proto.Marshal(&pb.Hello{})
	if err != nil {
		return nil, err
	}
	_, err = s.Write(b)
	if err != nil {
		return nil, err
	}
	rb := []byte{}
	n, err := s.Read(rb)
	if err != nil {
		return nil, err
	}

	// Parsing remote Hello
	{
		rHello := &pb.Hello{}
		proto.Unmarshal(rb[:n], rHello)
		// Does it require something we can't do ?
		if rHello.GetNeedStun() {
			// TODO: Send a Terminate message with the reason, also for other returns.
			return nil, err
		}
		// Send an empty STUN Server list if remote asked.
		if rHello.GetWantStun() || rHello.GetWantGoodStun() {
			b, err = proto.Marshal(&pb.StunShare{StunServers: []*pb.StunServer{}})
			if err != nil {
				return nil, err
			}
			_, err = s.Write(b)
			if err != nil {
				return nil, err
			}
		}
	}

	// Creating Offer
	{
		offer, err := pc.CreateOffer(nil)
		if err != nil {
			return nil, err
		}
		err = pc.SetLocalDescription(offer)
		if err != nil {
			return nil, err
		}
		b, err = proto.Marshal(&pb.SDP{Sdp: &offer.SDP})
		if err != nil {
			return nil, err
		}
		_, err = s.Write(b)
		if err != nil {
			return nil, err
		}
		n, err = s.Read(rb)
		if err != nil {
			return nil, err
		}
	}
	// Reading answer
	{
		rSdp := &pb.SDP{}
		proto.Unmarshal(rb[:n], rSdp)
		err = pc.SetRemoteDescription(webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  rSdp.GetSdp(),
		})
		if err != nil {
			return nil, err
		}
	}

	// Getting ICE candidates
	{
		icewg.Done()
		iceLocked = false
		// Checking if a connection worked.
		successChan := make(chan struct{})
		timeoutChan := make(chan struct{})
		pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			if state == webrtc.PeerConnectionStateConnected {
				select {
				case <-timeoutChan:
				case successChan <- struct{}{}:
				}
			}
		})
		go func() {
			time.Sleep(iceGraceTime)
			// Sending the ping genend message.
			b, err := proto.Marshal(
				&pb.WrapExchange{
					M: &pb.WrapExchange_Genend{
						Genend: &pb.GenerationEnded{},
					},
				},
			)
			if err != nil {
				return
			}
			icewg.Add(1)
			iceLocked = true
			s.Write(b)
			time.Sleep(connectionWaitTime)
			close(timeoutChan)
		}()
		var iceCount uint64 = 0
		for {
			n, err = s.Read(rb)
			if err != nil {
				return nil, err
			}
			// Parsing the message.
			rWSDP := &pb.WrapExchange{}
			proto.Unmarshal(rb[:n], rWSDP)
			// Checking message type.
			switch v := rWSDP.GetM().(type) {
			case *pb.WrapExchange_Terminate:
				return nil, err
			case *pb.WrapExchange_Ice:
				iceCount++
				// Adding the ICE candidate
				err = pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: v.Ice.GetIce()})
				if err != nil {
					return nil, err
				}
			case *pb.WrapExchange_Genend:
				if iceCount > 0 {
					break
				}
				// Terminating if no ICE candidate was received.
				return nil, err
			}
		}
		select {
		case <-successChan:
		case <-timeoutChan:
			// TODO: Implement retry
			return nil, ErrTimeout
		}
	}

	// Finishing
	endingBad = false
	return t.upgradeConn(pc, s), nil
}
