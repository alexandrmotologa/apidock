package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyExecution(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Test") != "apidock-proxy" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Server-Name", "test-backend")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"proxied"}`))
	}))
	defer ts.Close()

	client := NewClient()
	resp, err := client.Execute(context.Background(), Request{
		URL:    ts.URL,
		Method: http.MethodGet,
		Headers: map[string]string{
			"X-Custom-Test": "apidock-proxy",
		},
	})
	if err != nil {
		t.Fatalf("expected successful proxy execution, got error: %v", err)
	}

	if resp.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.Status)
	}

	if resp.Headers["X-Server-Name"] != "test-backend" {
		t.Errorf("expected header X-Server-Name to be test-backend, got %s", resp.Headers["X-Server-Name"])
	}

	if resp.Timing.TotalMs <= 0 {
		t.Errorf("expected positive total duration in ms, got %f", resp.Timing.TotalMs)
	}
}

func TestProxySSRFProtection(t *testing.T) {
	client := NewClient()
	_, err := client.Execute(context.Background(), Request{
		URL:    "http://169.254.169.254/latest/meta-data",
		Method: http.MethodGet,
	})
	if err == nil {
		t.Errorf("expected error for cloud metadata URL, got nil")
	}
}

func TestHistoryRecorder(t *testing.T) {
	recorder := NewHistoryRecorder(3)

	for i := 1; i <= 5; i++ {
		recorder.Record(
			Request{URL: fmt.Sprintf("https://api.example.com/%d", i)},
			Response{Status: 200},
		)
	}

	entries := recorder.GetAll()
	if len(entries) != 3 {
		t.Fatalf("expected capacity limited to 3 entries, got %d", len(entries))
	}

	// First entry should be the latest (5)
	if entries[0].Request.URL != "https://api.example.com/5" {
		t.Errorf("expected first entry to be request 5, got %s", entries[0].Request.URL)
	}

	recorder.Clear()
	if len(recorder.GetAll()) != 0 {
		t.Errorf("expected empty history after Clear()")
	}
}

func TestProxyExecuteStream(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if ok {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: chunk 1\n\n"))
		if ok {
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: chunk 2\n\n"))
		if ok {
			flusher.Flush()
		}
	}))
	defer ts.Close()

	client := NewClient()
	var receivedChunks []string
	var receivedStatus int

	err := client.ExecuteStream(context.Background(), Request{
		URL:    ts.URL,
		Method: http.MethodGet,
	}, StreamCallback{
		OnStart: func(status int, statusText string, headers map[string]string) error {
			receivedStatus = status
			return nil
		},
		OnChunk: func(chunk []byte) error {
			receivedChunks = append(receivedChunks, string(chunk))
			return nil
		},
	})

	if err != nil {
		t.Fatalf("unexpected streaming error: %v", err)
	}

	if receivedStatus != http.StatusOK {
		t.Errorf("expected status 200, got %d", receivedStatus)
	}

	combined := ""
	for _, c := range receivedChunks {
		combined += c
	}

	if combined != "data: chunk 1\n\ndata: chunk 2\n\n" {
		t.Errorf("unexpected streamed payload: %q", combined)
	}
}
