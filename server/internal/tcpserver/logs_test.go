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

// synced waits until the hub has handled what agent sent before.
func (h *testHub) synced(agent *testAgent) {
	agent.send(MsgSysStat, "", SystemStat{})
	<-h.chans.stat
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
