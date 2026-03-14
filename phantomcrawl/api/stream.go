package api

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
	streamClients   = map[chan string]bool{}
	streamClientsMu sync.Mutex
)

func BroadcastEvent(event string) {
	streamClientsMu.Lock()
	defer streamClientsMu.Unlock()
	for ch := range streamClients {
		select {
		case ch <- event:
		default:
		}
	}
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan string, 10)

	streamClientsMu.Lock()
	streamClients[ch] = true
	streamClientsMu.Unlock()

	defer func() {
		streamClientsMu.Lock()
		delete(streamClients, ch)
		streamClientsMu.Unlock()
		close(ch)
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Keep alive ping every 30s
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}