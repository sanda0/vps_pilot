package tcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type listResult struct {
	files LogFiles
	err   error
}

// list starts a List and returns the log_list the agent was sent.
func (h *testHub) list(agent *testAgent, nodeID int64) (LogList, <-chan listResult) {
	h.t.Helper()
	done := make(chan listResult, 1)
	go func() {
		files, err := h.logs.List(context.Background(), nodeID, "/var/www/shop")
		done <- listResult{files, err}
	}()
	var sent LogList
	json.Unmarshal(agent.expect(MsgLogList).Data, &sent)
	return sent, done
}

// open opens a stream while the agent reads the log_open it is sent.
func (h *testHub) open(agent *testAgent, nodeID int64) (LogOpen, <-chan LogEvent, func()) {
	h.t.Helper()
	type result struct {
		events      <-chan LogEvent
		closeStream func()
		err         error
	}
	done := make(chan result, 1)
	go func() {
		events, closeStream, err := h.logs.Open(OpenLog{
			NodeID: nodeID, ProjectPath: "/var/www/shop", File: "storage/logs/laravel.log", MaxBytes: 65536, Follow: true,
		})
		done <- result{events, closeStream, err}
	}()
	var sent LogOpen
	json.Unmarshal(agent.expect(MsgLogOpen).Data, &sent)
	res := <-done
	if res.err != nil {
		h.t.Fatal(res.err)
	}
	return sent, res.events, res.closeStream
}

// synced waits until the hub has handled what agent sent before.
func (h *testHub) synced(agent *testAgent) {
	agent.send(MsgSysStat, "", SystemStat{})
	<-h.chans.stat
}

func drain(events <-chan LogEvent) []LogEvent {
	var got []LogEvent
	for ev := range events {
		got = append(got, ev)
	}
	return got
}

func TestLogsNeedAnOnlineV04Agent(t *testing.T) {
	h := newTestHub(t)
	if _, err := h.logs.List(context.Background(), 99, "/var/www/shop"); !errors.Is(err, ErrNodeOffline) {
		t.Fatalf("offline node: %v", err)
	}
	_, nodeID := h.commandsAgent() // v0.3
	if _, err := h.logs.List(context.Background(), nodeID, "/var/www/shop"); !errors.Is(err, ErrNoLogs) {
		t.Fatalf("v0.3 agent list: %v", err)
	}
	if _, _, err := h.logs.Open(OpenLog{NodeID: nodeID}); !errors.Is(err, ErrNoLogs) {
		t.Fatalf("v0.3 agent open: %v", err)
	}
}

func TestLogListTakesTheAnswerFromItsNode(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureLogs)
	other, _ := h.agentWith(FeatureLogs)
	sent, done := h.list(agent, nodeID)
	if sent.ProjectPath != "/var/www/shop" || sent.RequestID == "" {
		t.Fatalf("sent %+v", sent)
	}

	other.send(MsgLogFiles, "", LogFiles{RequestID: sent.RequestID, Error: "from another node"})
	h.synced(other)
	files := []LogFile{{Path: "storage/logs/laravel.log", Size: 10, Modified: 1792915200}}
	agent.send(MsgLogFiles, "", LogFiles{RequestID: sent.RequestID, Files: files})
	res := <-done
	if res.err != nil || res.files.Error != "" || len(res.files.Files) != 1 || res.files.Files[0] != files[0] {
		t.Fatalf("got %+v, %v", res.files, res.err)
	}
}

func TestLogListTimesOutOrEndsOnDisconnect(t *testing.T) {
	h := newTestHub(t)
	old := LogListTimeout
	LogListTimeout = 50 * time.Millisecond
	t.Cleanup(func() { LogListTimeout = old })

	agent, nodeID := h.agentWith(FeatureLogs)
	if _, done := h.list(agent, nodeID); !errors.Is((<-done).err, ErrNoAnswer) {
		t.Fatal("no timeout")
	}
	LogListTimeout = time.Minute
	_, done := h.list(agent, nodeID)
	agent.conn.Close()
	if err := (<-done).err; !errors.Is(err, ErrNodeLeft) {
		t.Fatalf("after disconnect: %v", err)
	}
}

func TestLogStreamRelaysChunksThenEnd(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureLogs)
	sent, events, closeStream := h.open(agent, nodeID)
	defer closeStream()
	if sent.StreamID == "" || sent.File != "storage/logs/laravel.log" || sent.Before != nil || sent.MaxBytes != 65536 || !sent.Follow {
		t.Fatalf("sent %+v", sent)
	}

	agent.send(MsgLogData, "", LogData{StreamID: sent.StreamID, Seq: 0, Offset: 100, Data: "a\n", Size: 102})
	agent.send(MsgLogData, "", LogData{StreamID: sent.StreamID, Seq: 1, Offset: 0, Data: "b\n", Size: 2, Rotated: true})
	agent.send(MsgLogEnd, "", LogEnd{StreamID: sent.StreamID, Reason: "limit"})
	got := drain(events)
	if len(got) != 3 || got[0].Data.Data != "a\n" || !got[1].Data.Rotated || got[2].End.Reason != "limit" {
		t.Fatalf("events %+v", got)
	}
}

func TestLogStreamCloseTellsTheAgent(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureLogs)
	sent, events, closeStream := h.open(agent, nodeID)
	go closeStream()
	if got := agent.expect(MsgLogClose); string(got.Data) != `{"stream_id":"`+sent.StreamID+`"}` {
		t.Fatalf("close sent %s", got.Data)
	}
	if got := drain(events); len(got) != 1 || got[0].End.Reason != "closed" {
		t.Fatalf("events %+v", got)
	}
	// The agent's own log_end comes after; it's ignored.
	agent.send(MsgLogEnd, "", LogEnd{StreamID: sent.StreamID, Reason: "closed"})
	h.synced(agent)
	closeStream()
}

func TestLogStreamIgnoresOtherNodesAndEndsOnDisconnect(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureLogs)
	sent, events, closeStream := h.open(agent, nodeID)
	defer closeStream()

	other, _ := h.agentWith(FeatureLogs)
	other.send(MsgLogData, "", LogData{StreamID: sent.StreamID, Data: "x"})
	other.send(MsgLogEnd, "", LogEnd{StreamID: sent.StreamID, Reason: "done"})
	h.synced(other)

	agent.conn.Close()
	got := drain(events)
	if len(got) != 1 || got[0].End.Reason != "error" || got[0].End.Error != ErrNodeLeft.Error() {
		t.Fatalf("events %+v", got)
	}
}

func TestLogStreamClosedWhenReaderFallsBehind(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureLogs)
	sent, events, closeStream := h.open(agent, nodeID)
	defer closeStream()

	for i := 0; i <= streamBuffer; i++ {
		agent.send(MsgLogData, "", LogData{StreamID: sent.StreamID, Seq: int64(i), Data: "x"})
	}
	agent.expect(MsgLogClose)
	got := drain(events)
	if len(got) != streamBuffer+1 || got[streamBuffer].End.Error != errFellBehind {
		t.Fatalf("%d events, last %+v", len(got), got[len(got)-1])
	}
}

func TestLogEndWithUnknownReasonIsAnError(t *testing.T) {
	h := newTestHub(t)
	agent, nodeID := h.agentWith(FeatureLogs)
	sent, events, closeStream := h.open(agent, nodeID)
	defer closeStream()
	agent.send(MsgLogEnd, "", LogEnd{StreamID: sent.StreamID, Reason: "bored"})
	if got := drain(events); len(got) != 1 || got[0].End.Reason != "error" || got[0].End.Error == "" {
		t.Fatalf("events %+v", got)
	}
}
