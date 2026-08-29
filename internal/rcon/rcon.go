// Package rcon speaks Source RCON, which is how everything here asks the
// server a question.
//
// A packet at a time, one connection per command. The shell version shelled out
// to python for every line and could not tell a refused answer from an empty
// one, which is how a run measured a mission the server had never loaded.
package rcon

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

const (
	typeAuth        int32 = 3
	typeAuthReponse int32 = 2
	typeExec        int32 = 2
	typeResponse    int32 = 0
)

// ErrAuth is a password the server refused. Worth its own error: every other
// failure here is worth retrying and this one never is.
var ErrAuth = errors.New("rcon: the server refused the password")

// Client is one server. Dial per command rather than holding a connection: a
// map change drops it, and a run does a map change on purpose.
type Client struct {
	Addr     string
	Password string
	Timeout  time.Duration
}

// Do opens a connection, authenticates, sends one command and returns the
// reply. One connection per command: srcds drops an idle one and a reused
// connection failed in the middle of a run rather than at the start of one.
func (c Client) Do(command string) (string, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	conn, err := net.DialTimeout("tcp", c.Addr, timeout)
	if err != nil {
		return "", fmt.Errorf("rcon: cannot reach %s: %w", c.Addr, err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "", err
	}

	if err := write(conn, 1, typeAuth, c.Password); err != nil {
		return "", err
	}
	// The server answers an auth with an empty response and then the verdict.
	for {
		id, kind, _, err := read(conn)
		if err != nil {
			return "", err
		}
		if kind == typeAuthReponse {
			if id == -1 {
				return "", ErrAuth
			}
			break
		}
	}

	if err := write(conn, 2, typeExec, command); err != nil {
		return "", err
	}
	var out strings.Builder
	for {
		_, kind, body, err := read(conn)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			// A deadline here means the server said everything it was going to.
			var timeoutErr net.Error
			if errors.As(err, &timeoutErr) && timeoutErr.Timeout() && out.Len() > 0 {
				break
			}
			return out.String(), err
		}
		if kind != typeResponse {
			continue
		}
		out.WriteString(body)
		// One packet is the usual answer. Anything longer arrives in more, and
		// the short read below is what ends it.
		if len(body) < 4000 {
			break
		}
	}
	return out.String(), nil
}

/*
	write frames one packet

The conversions below are the wire format rather than arithmetic: Source's rcon
puts the id and the type in as little endian 32 bit words, and a negative id is
exactly how the protocol says an authentication failed. Reinterpreting the bits
is the encoding, so the overflow gosec warns about is the point.
*/
//nolint:gosec // G115: the conversions are the wire format, not arithmetic
func write(w io.Writer, id, kind int32, body string) error {
	payload := make([]byte, 0, len(body)+10)
	payload = binary.LittleEndian.AppendUint32(payload, uint32(id))
	payload = binary.LittleEndian.AppendUint32(payload, uint32(kind))
	payload = append(payload, body...)
	payload = append(payload, 0, 0)

	header := binary.LittleEndian.AppendUint32(nil, uint32(len(payload)))
	if _, err := w.Write(append(header, payload...)); err != nil {
		return fmt.Errorf("rcon: cannot send: %w", err)
	}
	return nil
}

//nolint:gosec // G115: the same wire format read back, and a negative id is how the protocol says the authentication failed
func read(r io.Reader) (id, kind int32, body string, err error) {
	var size int32
	if err = binary.Read(r, binary.LittleEndian, &size); err != nil {
		return 0, 0, "", err
	}
	if size < 10 || size > 1<<20 {
		return 0, 0, "", fmt.Errorf("rcon: a packet claiming %d bytes is not one", size)
	}
	buf := make([]byte, size)
	if _, err = io.ReadFull(r, buf); err != nil {
		return 0, 0, "", err
	}
	id = int32(binary.LittleEndian.Uint32(buf[0:4]))
	kind = int32(binary.LittleEndian.Uint32(buf[4:8]))
	return id, kind, string(buf[8 : len(buf)-2]), nil
}
