package tcpserver

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestJSONFrameRoundTrip(t *testing.T) {
	originalData := []byte(`{"cpu_usage":[12.5],"mem_usage":42}`)
	original := Msg{Msg: "sys_stat", NodeId: 7, Data: originalData}

	var buffer bytes.Buffer
	if err := writeJSONFrame(&buffer, original); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	decoded, err := readJSONFrame(&buffer)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if decoded.Msg != original.Msg || decoded.NodeId != original.NodeId {
		t.Fatalf("unexpected envelope: %#v", decoded)
	}
	if !json.Valid(decoded.Data) || !bytes.Equal(decoded.Data, originalData) {
		t.Fatalf("unexpected data: %s", decoded.Data)
	}
}

func TestJSONFrameRejectsOversizedPayload(t *testing.T) {
	var buffer bytes.Buffer
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], maxAgentFrameSize+1)
	buffer.Write(header[:])

	if _, err := readJSONFrame(&buffer); err == nil {
		t.Fatal("expected oversized frame to be rejected")
	}
}

func TestRequireAgentToken(t *testing.T) {
	called := false
	handler := RequireAgentToken("expected")(func(_ context.Context, _ *ConnContext, _ Msg) {
		called = true
	})

	handler(context.Background(), &ConnContext{RemoteAddr: "test"}, Msg{Token: "wrong"})
	if called {
		t.Fatal("handler ran with an invalid token")
	}

	handler(context.Background(), &ConnContext{RemoteAddr: "test"}, Msg{Token: "expected"})
	if !called {
		t.Fatal("handler did not run with the valid token")
	}
}
