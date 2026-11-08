package scanner

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type clamavScanner struct {
	address string
	timeout time.Duration
}

func NewClamAVScanner(host, port string, timeout time.Duration) *clamavScanner {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &clamavScanner{
		address: net.JoinHostPort(host, port),
		timeout: timeout,
	}

}
func (c *clamavScanner) dial(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, fmt.Errorf("clamav dial %s: %w", c.address, err)
	}
	return conn, nil
}

// Ping checks if the ClamAV server is reachable (Health Check)
func (c *clamavScanner) Ping(ctx context.Context) error {
	conn, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("zPING\x00")); err != nil {
		return fmt.Errorf("clamav.Ping: failed to send PING: %w", err)
	}
	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\x00')
	if err != nil {
		return fmt.Errorf("clamav.Ping: failed to read response: %w", err)
	}
	if !strings.Contains(resp, "PONG") {
		return fmt.Errorf("clamav.Ping: unexpected response: %s", resp)
	}
	return nil
}

// Scan streams data directly to ClamAV via zINSTREAM command
func (c *clamavScanner) Scan(ctx context.Context, r io.Reader) (*ScanResult, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// Send zINSTREAM command terminated with a null byte
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return nil, fmt.Errorf("clamav.Scan: failed to initiate zINSTREAM: %w", err)
	}
	// Stream chunks of 32KB to ClamAV
	buf := make([]byte, 32*1024)
	lenBuf := make([]byte, 4)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		n, readErr := r.Read(buf)
		if n > 0 {
			// INSTREAM format: [4 bytes chunk size (big-endian)] + [data]
			binary.BigEndian.PutUint32(lenBuf, uint32(n))
			if _, err := conn.Write(lenBuf); err != nil {
				return nil, fmt.Errorf("clamav.Scan: failed to write chunk size: %w", err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return nil, fmt.Errorf("clamav.Scan: failed to write chunk data: %w", err)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return nil, fmt.Errorf("clamav.Scan: error reading stream: %w", readErr)
		}
	}
	// Send 0-sized chunk to signal end of stream
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return nil, fmt.Errorf("clamav.Scan: failed to terminate stream: %w", err)
	}
	// Read response from ClamAV
	// Responses typically format as:
	// "stream: OK\x00" or "stream: Eicar-Signature FOUND\x00"
	reader := bufio.NewReader(conn)
	resp, err := reader.ReadString('\x00')
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("clamav.Scan: failed to read scan response: %w", err)
	}
	resp = strings.TrimSpace(strings.Trim(resp, "\x00"))
	if strings.HasSuffix(resp, "OK") {
		return &ScanResult{IsClean: true}, nil
	}
	if strings.HasSuffix(resp, "FOUND") {
		// Parse virus name: "stream: <VirusName> FOUND"
		parts := strings.Split(resp, ": ")
		virusName := "Unknown"
		if len(parts) >= 2 {
			virusName = strings.TrimSuffix(parts[1], " FOUND")
		}
		return &ScanResult{
			IsClean:   false,
			VirusName: virusName,
		}, nil
	}
	return nil, fmt.Errorf("clamav.Scan: scan error from daemon: %s", resp)
}
