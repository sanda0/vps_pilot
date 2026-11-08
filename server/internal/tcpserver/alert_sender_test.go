package tcpserver

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testAlert() AlertMsg {
	return AlertMsg{
		NodeName:     "web-01",
		NodeIp:       "203.0.113.10",
		Metric:       "cpu",
		Threshold:    "90%",
		CurrentValue: "98%",
		Timestamp:    time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
}

func TestSendDiscordAlertPostsJSON(t *testing.T) {
	var method, contentType, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		contentType = r.Header.Get("Content-Type")
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		body = string(raw)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	alert := testAlert()
	if err := SendDiscordAlert(srv.URL, alert); err != nil {
		t.Fatalf("SendDiscordAlert() error = %v", err)
	}

	if method != http.MethodPost {
		t.Errorf("method = %q, want %s", method, http.MethodPost)
	}
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("request body is not a JSON object: %v", err)
	}
	content, ok := payload["content"]
	if !ok {
		t.Fatalf("payload has no content field: %s", body)
	}
	for _, want := range []string{alert.NodeName, alert.Metric, alert.CurrentValue} {
		if !strings.Contains(content, want) {
			t.Errorf("content %q does not mention %q", content, want)
		}
	}
}

func TestSendSlackAlertPostsJSON(t *testing.T) {
	var method, contentType, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		contentType = r.Header.Get("Content-Type")
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		body = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	alert := testAlert()
	if err := SendSlackAlert(srv.URL, alert); err != nil {
		t.Fatalf("SendSlackAlert() error = %v", err)
	}

	if method != http.MethodPost {
		t.Errorf("method = %q, want %s", method, http.MethodPost)
	}
	if !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("request body is not a JSON object: %v", err)
	}
	text, ok := payload["text"].(string)
	if !ok {
		t.Fatalf("payload has no text field: %s", body)
	}
	for _, want := range []string{alert.NodeName, alert.Metric, alert.CurrentValue} {
		if !strings.Contains(text, want) {
			t.Errorf("text %q does not mention %q", text, want)
		}
	}
	if _, ok := payload["attachments"]; !ok {
		t.Errorf("payload has no attachments field: %s", body)
	}
}

func TestAlertSendersReturnErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	senders := []struct {
		name string
		send func(string, AlertMsg) error
	}{
		{"discord", SendDiscordAlert},
		{"slack", SendSlackAlert},
	}
	for _, sender := range senders {
		t.Run(sender.name, func(t *testing.T) {
			err := sender.send(srv.URL, testAlert())
			if err == nil {
				t.Fatal("want error for 500 response, got nil")
			}
			if !strings.Contains(err.Error(), "500") {
				t.Errorf("error = %v, want the status code in the message", err)
			}
		})
	}
}

func TestAlertSendersTimeOut(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	previousTimeout := alertWebhookTimeout
	alertWebhookTimeout = 20 * time.Millisecond
	defer func() { alertWebhookTimeout = previousTimeout }()

	senders := []struct {
		name string
		send func(string, AlertMsg) error
	}{
		{"discord", SendDiscordAlert},
		{"slack", SendSlackAlert},
	}
	for _, sender := range senders {
		t.Run(sender.name, func(t *testing.T) {
			start := time.Now()
			err := sender.send(srv.URL, testAlert())
			elapsed := time.Since(start)

			if err == nil {
				t.Fatal("want a timeout error, got nil")
			}
			if elapsed >= 200*time.Millisecond {
				t.Errorf("call took %v, want it bounded by the %v client timeout", elapsed, alertWebhookTimeout)
			}
		})
	}
}
