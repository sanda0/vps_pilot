package tcpserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := Msg{Msg: MsgSysStat, NodeId: 7, Token: "t", Data: json.RawMessage(`{"mem_usage":1.5}`)}
	if err := WriteFrame(&buf, in); err != nil {
		t.Fatal(err)
	}

	var out Msg
	if err := ReadFrame(&buf, &out); err != nil {
		t.Fatal(err)
	}
	if out.Msg != in.Msg || out.NodeId != in.NodeId || out.Token != in.Token || string(out.Data) != string(in.Data) {
		t.Fatalf("got %+v, want %+v", out, in)
	}
}

func TestReadFrameRejectsBadSizes(t *testing.T) {
	for _, size := range []uint32{0, MaxFrameSize + 1} {
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, size)
		var msg Msg
		if err := ReadFrame(&buf, &msg); err == nil {
			t.Errorf("size %d: expected error", size)
		}
	}
}
