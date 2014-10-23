package gosockets

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Hub struct {
	connections map[*Connection]bool
	write       chan []byte
	register    chan *Connection
	unregister  chan *Connection
	events      map[int]func(*Connection, []byte)
	upgrader    *websocket.Upgrader
}

func NewHub(origin func(r *http.Request) bool) *Hub {
	if origin == nil {
		origin = func(r *http.Request) bool {
			return true
		}
	}
	hub := &Hub{
		connections: make(map[*Connection]bool),
		write:       make(chan []byte),
		register:    make(chan *Connection),
		unregister:  make(chan *Connection),
		events:      make(map[int]func(*Connection, []byte)),
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     origin,
		},
	}
	go hub.run()
	return hub
}

func (h *Hub) AddEvent(event int, fn func(*Connection, []byte)) {
	h.events[event] = fn
}

func (h *Hub) RemoveEvent(event int) {
	delete(h.events, event)
}

func (h *Hub) Broadcast(message []byte) {
	for c := range h.connections {
		c.Write(message)
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	c := NewConnection(h, ws)
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writer()
	c.reader()
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.connections[c] = true
		case c := <-h.unregister:
			if _, ok := h.connections[c]; ok {
				delete(h.connections, c)
				close(c.write)
			}
		case m := <-h.write:
			for c := range h.connections {
				select {
				case c.write <- m:
				default:
					delete(h.connections, c)
					close(c.write)
				}
			}
		}
	}
}
