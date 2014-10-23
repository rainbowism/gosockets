package gosockets

import (
	"github.com/gorilla/websocket"
)

type Connection struct {
	h     *Hub
	ws    *websocket.Conn
	write chan []byte
}

func NewConnection(h *Hub, ws *websocket.Conn) *Connection {
	return &Connection{
		h:     h,
		write: make(chan []byte, 256),
		ws:    ws,
	}
}

func (c *Connection) Write(message []byte) {
	c.write <- message
}

func (c *Connection) reader() {
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			break
		}

		for _, fn := range c.h.events {
			fn(c, message)
		}
	}
}

func (c *Connection) writer() {
	for message := range c.write {
		err := c.ws.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			break
		}
	}
	c.ws.Close()
}
