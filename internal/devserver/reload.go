package devserver

import (
	"fmt"
	"net/http"
	"sync"
)

const liveReloadPath = "/.margo/live-reload"

const liveReloadClient = `<script data-margo-development-live-reload>(function(){
  var events = new EventSource("/.margo/live-reload");
  events.addEventListener("reload", function(){ location.reload(); });
})();</script>`

// Broker publishes successful build generations to connected browsers.
type Broker struct {
	mu          sync.Mutex
	generation  uint64
	nextID      uint64
	subscribers map[uint64]chan uint64
}

// NewBroker creates an empty live-reload broker.
func NewBroker() *Broker {
	return &Broker{subscribers: make(map[uint64]chan uint64)}
}

// Publish advances the visible build generation and notifies subscribers.
func (broker *Broker) Publish(generation uint64) {
	if broker == nil {
		return
	}
	broker.mu.Lock()
	broker.generation = generation
	for _, subscriber := range broker.subscribers {
		select {
		case subscriber <- generation:
		default:
		}
	}
	broker.mu.Unlock()
}

// ServeHTTP streams ready and reload events using Server-Sent Events.
func (broker *Broker) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Connection", "keep-alive")

	broker.mu.Lock()
	id := broker.nextID
	broker.nextID++
	updates := make(chan uint64, 1)
	broker.subscribers[id] = updates
	generation := broker.generation
	broker.mu.Unlock()
	defer func() {
		broker.mu.Lock()
		delete(broker.subscribers, id)
		broker.mu.Unlock()
	}()

	fmt.Fprintf(writer, "event: ready\ndata: %d\n\n", generation)
	flusher.Flush()
	for {
		select {
		case generation := <-updates:
			fmt.Fprintf(writer, "event: reload\ndata: %d\n\n", generation)
			flusher.Flush()
		case <-request.Context().Done():
			return
		}
	}
}
