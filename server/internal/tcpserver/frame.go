package tcpserver

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// MaxFrameSize caps a single frame so a peer can't make us allocate unbounded memory.
const MaxFrameSize = 1 << 20

// ReadFrame reads one length-prefixed JSON frame: a 4-byte big-endian
// payload length followed by a JSON-encoded Msg.
func ReadFrame(r io.Reader, msg *Msg) error {
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(header[:])
	if size == 0 || size > MaxFrameSize {
		return fmt.Errorf("invalid frame size %d", size)
	}
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return err
	}
	if err := json.Unmarshal(buf, msg); err != nil {
		return fmt.Errorf("decode frame: %w", err)
	}
	return nil
}

// WriteFrame writes msg as one length-prefixed JSON frame.
func WriteFrame(w io.Writer, msg Msg) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(body) > MaxFrameSize {
		return fmt.Errorf("frame too large: %d bytes", len(body))
	}
	buf := make([]byte, 4+len(body))
	binary.BigEndian.PutUint32(buf, uint32(len(body)))
	copy(buf[4:], body)
	_, err = w.Write(buf)
	return err
}
