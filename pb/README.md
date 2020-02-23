# Aside exchange protocol

This is a just a way to exchange the SDP and negociate the protocol.

It also implement an STUN server sharing to allow better implementation of DHT
addressed STUN.

## Hello

When a new connection is created each nodes send an hello message to the other,
this message contains a list of uint8, the first 7 bit represent the id of the
protocol to be used (table found in [exchange.proto](./exchange.proto)) the last
bit indicate if trickle can be used (1 yes, 0 no).

The order of the list represent the want of nodes to use this protocol, smaller
index is bigger. To avoid a confirmation of the negociated protocol only the
want of the initiator are important, the receptor list is only here to check if
he support the protocol.

There is also some bool values about stun exchange:
- Share STUN, if true indicate this node is willing to provide stun servers.
- Share GOOD STUN, if true indicate this node is willing to provide known good
stun servers.
- Need STUN, if true the node need some STUN servers, if the other node is not
able to provide some the connection will be aborted.
- Want STUN, if true the node may want some STUN servers (have some but no good
known one).
- Want GOOD STUN, if true the node want some known good STUN servers.

## STUN Share

List of (string, uint16, bool) tuples representing a stun server (domain
(without `stun:`), port, known good ?). If the ip is a private ip its gonna
terminate the connection. If someone indicated any kind of want and the other
node is capable to share a stun share message is needed to proceed further in
the exchange.

## Exchange

Once the initiator have received the hello message he can start sending SDP
offers. Then the receptor can answer with an answer, assume with role (initiator
offer).

If trickling the message must be wrapped with WrapExchange (to differenciate the
message type).

### Proto implementation note

Each implementation of WebRTC Aside Transport should at least support
DataChannel proto because acording to [caniuse.com](https://caniuse.com/)
DataChannel is the most supported WebRTC bytes like API on browser. Its
really important because libp2p doesn't expect a negociation to fail.

## Status

Status messages are used to keep track of where nodes are in the process of
handshake.
They are empty messages, status are carried in message id.

### Generation ended (TRICKLE)

This message indicate that no more SDP are to expect. So if none worked
currently may terminate exchange.

The initiator send this right after the last SDP were sended (or not (STUN
timeout)). The receptor wait for the initiator to send his `Generation ended`
first (to prevent missing later SDP).

If trickle is not enabled the SDP offer or answer count as `Generation ended`.

If trickling the message must be wrapped with WrapExchange (to differenciate the
message type).

### Answer testing ended (initiator)

When the initiator have finished to test all answers he send this indicating
exchange may end.

Some bool values:
- retry, if true this indicate the initiator is willing to retry
with other STUN servers if the connection failed.
- More stun, if true the initiator want more STUN server.
- More good stun, if true the initiator want more STUN server.

The message must be wrapped with WrapAnswerEnd (to differenciate the message
type).

### Retry accept (receptor)

After the `Answer testing ended` if accepting to retry.

Some bool values:
- More stun, if true the receptor want more STUN server.

Then the comunication jump to STUN share again.

Implementations may have a limit amount of retry (3 is a good number) and then
terminating with `Ran out of retries`.

The message must be wrapped with WrapAnswerEnd (to differenciate the message
type).

### Terminate.

Terminate, include an reason id :
- Worked.
- Ran out of STUN servers.
- Ran out of retries.
- Private ip stun server.
