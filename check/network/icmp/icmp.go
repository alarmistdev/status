package icmp

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/alarmistdev/status/check"
)

// Check creates a health check for ICMP ping
func Check(host string) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		// Create a context with timeout
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		// Create a channel to receive the result
		done := make(chan error, 1)

		go func() {
			// Create a new ICMP connection
			conn, err := net.DialTimeout("ip4:icmp", host, 5*time.Second)
			if err != nil {
				done <- fmt.Errorf("failed to create ICMP connection: %w", err)
				return
			}
			defer conn.Close()

			// Set deadline for the connection
			if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
				done <- fmt.Errorf("failed to set deadline: %w", err)
				return
			}

			// Create ICMP echo request message
			// Type 8 = Echo Request, Code 0
			// Identifier and Sequence number for matching request/reply
			// Data payload for echo
			msg := []byte{
				8, 0, 0, 0, // Type, Code, Checksum (to be filled)
				0, 1, // Identifier
				0, 1, // Sequence number
				0, 0, 0, 0, // Timestamp
				0, 0, 0, 0, // Data payload
			}

			// Calculate checksum
			checksum := calculateChecksum(msg)
			msg[2] = byte(checksum >> 8)
			msg[3] = byte(checksum & 0xff)

			if _, err := conn.Write(msg); err != nil {
				done <- fmt.Errorf("failed to send ICMP echo request: %w", err)
				return
			}

			// Read response
			reply := make([]byte, 1024)
			n, err := conn.Read(reply)
			if err != nil {
				done <- fmt.Errorf("failed to receive ICMP echo reply: %w", err)
				return
			}

			// Validate reply
			if n < 8 {
				done <- fmt.Errorf("invalid ICMP reply length: %d", n)
				return
			}

			// Check if it's an echo reply (type 0)
			if reply[0] != 0 {
				done <- fmt.Errorf("unexpected ICMP reply type: %d", reply[0])
				return
			}

			// Verify identifier and sequence number match
			if reply[4] != msg[4] || reply[5] != msg[5] || reply[6] != msg[6] || reply[7] != msg[7] {
				done <- fmt.Errorf("ICMP reply identifier/sequence mismatch")
				return
			}

			done <- nil
		}()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	})
}

// calculateChecksum calculates the ICMP checksum
func calculateChecksum(msg []byte) uint16 {
	var sum uint32
	for i := 0; i < len(msg); i += 2 {
		if i+1 < len(msg) {
			sum += uint32(msg[i])<<8 | uint32(msg[i+1])
		} else {
			sum += uint32(msg[i]) << 8
		}
	}
	sum = (sum >> 16) + (sum & 0xffff)
	sum += sum >> 16
	return ^uint16(sum)
}
