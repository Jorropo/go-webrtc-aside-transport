
# Go webrtc aside transport

***WIP*** This transport is currently under devlopement, features may not work at
all.

This transport is a [webrtc wrapper](https://github.com/pion/webrtc) for [libp2p](https://github.com/libp2p/go-libp2p).

This transport uses webrtc dtls for security, and sctp for muxing layer.

This transport is fully avaible in the browser using WASM, an js wrapper will
come.

This transport aim to allow high quality desktop <-> browser,
browser <-> browser connection.

## SDP Exchange

The exchange of SDP is made using trickle through an libp2p stream, so to dial
an webrtc aside transport libp2p first dial through an transport such as ws or
[circuit](https://github.com/libp2p/go-libp2p-circuit), then the connection could be used to build an webrtc udp faster one.

## STUN Server

There is currently 2 way for finding STUN server:
- Pre known, these are simply in the config.
- DHT (not even drafted), some libp2p nodes publicly accesible on internet will
  run integrated STUN server and advertise them in the dht.
