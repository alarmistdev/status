package icmp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/alarmistdev/status/check"
)

const (
	defaultICMPTimeout    = 5 * time.Second
	icmpEchoRequestType   = 8
	icmpEchoReplyType     = 0
	icmpIdentifier        = 1
	icmpSequence          = 1
	icmpHeaderLength      = 8
	icmpPayloadBufferSize = 1024
	checksumMask          = 0xffff
	byteOffset            = 8
	halfWordOffset        = 16
	lastByteMask          = 0xff
)

// Check creates a health check for ICMP ping.
func Check(host string) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, defaultICMPTimeout)
		defer cancel()

		return awaitPingResult(ctx, host)
	})
}

func awaitPingResult(ctx context.Context, host string) error {
	done := make(chan error, 1)

	go func() {
		done <- sendICMPEcho(ctx, host)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("ping context: %w", ctx.Err())
	}
}

func sendICMPEcho(ctx context.Context, host string) error {
	dialer := &net.Dialer{Timeout: defaultICMPTimeout}

	conn, err := dialer.DialContext(ctx, "ip4:icmp", host)
	if err != nil {
		return fmt.Errorf("failed to create ICMP connection: %w", err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(defaultICMPTimeout)); err != nil {
		return fmt.Errorf("failed to set deadline: %w", err)
	}

	msg := buildEchoRequest()
	if _, err := conn.Write(msg); err != nil {
		return fmt.Errorf("failed to send ICMP echo request: %w", err)
	}

	reply := make([]byte, icmpPayloadBufferSize)
	n, err := conn.Read(reply)
	if err != nil {
		return fmt.Errorf("failed to receive ICMP echo reply: %w", err)
	}

	return validateReply(msg, reply, n)
}

func buildEchoRequest() []byte {
	msg := []byte{
		icmpEchoRequestType, 0, 0, 0, // Type, Code, Checksum placeholder
		0, icmpIdentifier, // Identifier
		0, icmpSequence, // Sequence number
		0, 0, 0, 0, // Timestamp placeholder
		0, 0, 0, 0, // Data payload
	}

	checksum := calculateChecksum(msg)
	msg[2] = byte(checksum >> byteOffset)
	msg[3] = byte(checksum & lastByteMask)

	return msg
}

func validateReply(msg, reply []byte, replyLength int) error {
	if replyLength < icmpHeaderLength {
		return fmt.Errorf("invalid ICMP reply length: %d", replyLength)
	}

	if reply[0] != icmpEchoReplyType {
		return fmt.Errorf("unexpected ICMP reply type: %d", reply[0])
	}

	if !matchesIdentifiers(msg, reply) {
		return errors.New("ICMP reply identifier/sequence mismatch")
	}

	return nil
}

func matchesIdentifiers(request, reply []byte) bool {
	return reply[4] == request[4] &&
		reply[5] == request[5] &&
		reply[6] == request[6] &&
		reply[7] == request[7]
}

// calculateChecksum calculates the ICMP checksum.
func calculateChecksum(msg []byte) uint16 {
	var sum uint32
	for i := 0; i < len(msg); i += 2 {
		if i+1 < len(msg) {
			sum += uint32(msg[i])<<byteOffset | uint32(msg[i+1])
		} else {
			sum += uint32(msg[i]) << byteOffset
		}
	}
	sum = (sum >> halfWordOffset) + (sum & checksumMask)
	sum += sum >> halfWordOffset

	return ^uint16(sum)
}
