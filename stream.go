package aside

import (
	"io"
	"time"

	"github.com/pion/webrtc/v2"
)

type webRTCAsideStream struct {
	io.ReadWriteCloser
}

func upgradeDataChannel(d *webrtc.DataChannel) (*webRTCAsideStream, error) {
	dd, err := d.Detach()
	if err != nil {
		return nil, err
	}
	return &webRTCAsideStream{ReadWriteCloser: dd}, nil
}

func (s *webRTCAsideStream) Reset() error {
	return s.Close()
}

// TODO: Implement
func (_ *webRTCAsideStream) SetDeadline(_ time.Time) error {
	return nil
}

// TODO: Implement
func (_ *webRTCAsideStream) SetReadDeadline(_ time.Time) error {
	return nil
}

// TODO: Implement
func (_ *webRTCAsideStream) SetWriteDeadline(_ time.Time) error {
	return nil
}
