package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sanda0/vps_pilot/internal/db"
	"github.com/sanda0/vps_pilot/internal/services"
	"github.com/sanda0/vps_pilot/internal/tcpserver"
)

type fakeLogReader struct {
	err    error
	files  tcpserver.LogFiles
	opened []tcpserver.OpenLog
	events chan tcpserver.LogEvent
	closed chan struct{}
}

func (f *fakeLogReader) List(ctx context.Context, nodeID int64, projectPath string) (tcpserver.LogFiles, error) {
	return f.files, f.err
}

func (f *fakeLogReader) Open(req tcpserver.OpenLog) (<-chan tcpserver.LogEvent, func(), error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	f.opened = append(f.opened, req)
	return f.events, func() { close(f.closed) }, nil
}

func newLogTestServer(t *testing.T) (*gin.Engine, *fakeLogReader) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	op, ts, err := db.InitializeDatabases(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { op.Close(); ts.Close() })
	for _, q := range []string{
		`INSERT INTO nodes (id, name) VALUES (1, 'web')`,
		`INSERT INTO projects (id, node_id, name, path, logs) VALUES ('p1', 1, 'shop', '/var/www/shop', '["storage/logs"]')`,
		`INSERT INTO projects (id, node_id, name, path) VALUES ('p2', 1, 'blog', '/var/www/blog')`,
	} {
		if _, err := op.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	reader := &fakeLogReader{events: make(chan tcpserver.LogEvent, 8), closed: make(chan struct{})}
	h := NewLogHandler(services.NewLogService(context.Background(), db.NewRepo(op, ts), reader))
	r := gin.New()
	r.GET("/projects/:id/logs", h.List)
	r.GET("/projects/:id/logs/ws", h.Watch)
	return r, reader
}

func TestLogList(t *testing.T) {
	r, reader := newLogTestServer(t)
	for path, want := range map[string]int{"/projects/nope/logs": http.StatusNotFound, "/projects/p2/logs": http.StatusConflict} {
		if w := do(r, "GET", path, ""); w.Code != want {
			t.Errorf("GET %s: %d %s, want %d", path, w.Code, w.Body, want)
		}
	}

	if w := do(r, "GET", "/projects/p1/logs", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"files":[]`) {
		t.Fatalf("no files: %d %s", w.Code, w.Body)
	}
	reader.files = tcpserver.LogFiles{Files: []tcpserver.LogFile{{Path: "storage/logs/laravel.log", Size: 3}}}
	if w := do(r, "GET", "/projects/p1/logs", ""); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"path":"storage/logs/laravel.log"`) {
		t.Fatalf("files: %d %s", w.Code, w.Body)
	}
	for err, want := range map[error]int{tcpserver.ErrNoLogs: http.StatusConflict, tcpserver.ErrNoAnswer: http.StatusGatewayTimeout} {
		reader.err = err
		if w := do(r, "GET", "/projects/p1/logs", ""); w.Code != want {
			t.Errorf("%v: %d, want %d", err, w.Code, want)
		}
	}
}

func TestLogWebSocketRejectsBadRequests(t *testing.T) {
	r, reader := newLogTestServer(t)
	for _, q := range []string{"", "file=a.log&before=-1", "file=a.log&max_bytes=0", "file=a.log&before=10&follow=true"} {
		if w := do(r, "GET", "/projects/p1/logs/ws?"+q, ""); w.Code != http.StatusBadRequest {
			t.Errorf("?%s: %d %s", q, w.Code, w.Body)
		}
	}
	if w := do(r, "GET", "/projects/p2/logs/ws?file=a.log", ""); w.Code != http.StatusConflict {
		t.Errorf("no log folders: %d", w.Code)
	}
	if len(reader.opened) != 0 {
		t.Fatalf("opened %+v", reader.opened)
	}
}

func TestLogWebSocket(t *testing.T) {
	r, reader := newLogTestServer(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	url := "ws" + srv.URL[len("http"):] + "/projects/p1/logs/ws?file=storage/logs/laravel.log&before=100&max_bytes=10"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if got := reader.opened; len(got) != 1 || got[0].NodeID != 1 || got[0].ProjectPath != "/var/www/shop" ||
		got[0].File != "storage/logs/laravel.log" || *got[0].Before != 100 || got[0].MaxBytes != 10 || got[0].Follow {
		t.Fatalf("opened %+v", got)
	}

	reader.events <- tcpserver.LogEvent{Data: &tcpserver.LogData{Offset: 90, Data: "a\n"}}
	reader.events <- tcpserver.LogEvent{End: &tcpserver.LogEnd{Reason: "done"}}
	var got []string
	for {
		var ev tcpserver.LogEvent
		if err := conn.ReadJSON(&ev); err != nil {
			break
		}
		if ev.Data != nil {
			got = append(got, "data:"+ev.Data.Data)
		} else {
			got = append(got, "end:"+ev.End.Reason)
		}
	}
	if strings.Join(got, ",") != "data:a\n,end:done" {
		t.Fatalf("messages %q", got)
	}
	<-reader.closed
}

func TestLogWebSocketCloseClosesTheStream(t *testing.T) {
	r, reader := newLogTestServer(t)
	srv := httptest.NewServer(r)
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[len("http"):]+"/projects/p1/logs/ws?file=a.log&follow=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reader.opened[0].Follow || reader.opened[0].Before != nil || reader.opened[0].MaxBytes != defaultLogWindow {
		t.Fatalf("opened %+v", reader.opened)
	}
	conn.Close()
	<-reader.closed
}
