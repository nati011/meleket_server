package notification

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gotify/server/v2/model"
)

const (
	writeWait = 2 * time.Second
)

var ping = func(conn *websocket.Conn) error {
	return conn.WriteMessage(websocket.PingMessage, nil)
}

var writeJSON = func(conn *websocket.Conn, v interface{}) error {
	return conn.WriteJSON(v)
}

type client struct {
	conn    *websocket.Conn
	onClose func(*client)
	write   chan *model.NotificationMessageExternal
	userID  uint
	token   string
	once    once
}

func newClient(conn *websocket.Conn, userID uint, token string, onClose func(*client)) *client {
	return &client{
		conn:    conn,
		write:   make(chan *model.NotificationMessageExternal, 1),
		userID:  userID,
		token:   token,
		onClose: onClose,
	}
}

func (c *client) Close() {
	c.once.Do(func() {
		c.conn.Close()
		close(c.write)
	})
}

func (c *client) NotifyClose() {
	c.once.Do(func() {
		c.conn.Close()
		close(c.write)
		c.onClose(c)
	})
}

func (c *client) startReading(pongWait time.Duration) {
	defer c.NotifyClose()
	c.conn.SetReadLimit(64)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(appData string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.NextReader(); err != nil {
			printWebSocketError("ReadError", err)
			return
		}
	}
}

func (c *client) startWriteHandler(pingPeriod time.Duration) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		c.NotifyClose()
		pingTicker.Stop()
	}()

	for {
		select {
		case message, ok := <-c.write:
			if !ok {
				return
			}

			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := writeJSON(c.conn, message); err != nil {
				printWebSocketError("WriteError", err)
				return
			}
		case <-pingTicker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := ping(c.conn); err != nil {
				printWebSocketError("PingError", err)
				return
			}
		}
	}
}

func printWebSocketError(prefix string, err error) {
	closeError, ok := err.(*websocket.CloseError)

	if ok && closeError != nil && (closeError.Code == 1000 || closeError.Code == 1001) {
		// normal closure
		return
	}

	fmt.Println("WebSocket:", prefix, err)
}
