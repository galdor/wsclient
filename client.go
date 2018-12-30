package main

import (
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type Message struct {
	Type int
	Data []byte
}

type Client struct {
	URI  *url.URL
	Conn *websocket.Conn

	SendChan  chan string
	RecvChan  chan Message
	ErrorChan chan error

	wg       sync.WaitGroup
	stopChan chan struct{}
}

func NewClient(uri *url.URL) *Client {
	c := Client{
		URI: uri,

		SendChan:  make(chan string),
		RecvChan:  make(chan Message),
		ErrorChan: make(chan error),

		stopChan: make(chan struct{}),
	}

	return &c
}

func (c *Client) Start() error {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(c.URI.String(), nil)
	if err != nil {
		return err
	}
	c.Conn = conn

	c.wg.Add(2)
	go c.reader()
	go c.writer()

	return nil
}

func (c *Client) Stop() {
	if c.Conn == nil {
		return
	}

	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	deadline := time.Now().Add(250 * time.Millisecond)
	c.Conn.WriteControl(websocket.CloseMessage, msg, deadline)

	close(c.stopChan)
	c.Conn.Close()
	c.wg.Wait()

	close(c.SendChan)
	close(c.RecvChan)
	close(c.ErrorChan)
}

func (c *Client) Closing() bool {
	select {
	case <-c.stopChan:
		return true
	default:
	}

	return false
}

func (c *Client) reader() {
	defer c.wg.Done()

	for {
		msgType, data, err := c.Conn.ReadMessage()
		if err != nil {
			if c.Closing() {
				return
			}

			err2 := errors.Wrap(err, "cannot read message")
			c.ErrorChan <- err2
			return
		}

		c.RecvChan <- Message{
			Type: msgType,
			Data: data,
		}
	}
}

func (c *Client) writer() {
	defer c.wg.Done()

	for {
		select {
		case s := <-c.SendChan:
			if err := c.sendTextMessage(s); err != nil {
				if c.Closing() {
					return
				}

				err2 := errors.Wrap(err, "cannot send message")
				c.ErrorChan <- err2
				return
			}

		case <-c.stopChan:
			return
		}
	}
}

func (c *Client) sendTextMessage(s string) error {
	return c.Conn.WriteMessage(websocket.TextMessage, []byte(s))
}
