package kubeagent

import (
	"bufio"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
)

func NewRequestTransit(req *http.Request) *RequestTransit {
	return &RequestTransit{Request: req, ResponseOnce: make(chan *http.Response)}
}

type RequestTransit struct {
	*http.Request
	ResponseOnce chan *http.Response
}

func (r *RequestTransit) Dispatch(c *websocket.Conn) error {
	w, err := c.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	if err := r.Write(w); err != nil {
		return err
	}

	if err := w.Close(); err != nil {
		return err
	}

	return nil
}

func (r *RequestTransit) Wait(c *websocket.Conn) error {
	defer func() {
		close(r.ResponseOnce)
	}()

	_, respReader, err := c.NextReader()
	if err != nil {
		return err
	}

	resp, err := http.ReadResponse(bufio.NewReader(respReader), r.Request)
	if err != nil {
		return err
	}

	resp.Body = &ReaderCloser{
		Reader: resp.Body,
		Closes: []CloseFn{
			resp.Body.Close,
			c.Close,
		},
	}

	//defer func() {
	//	if err := c.Close(); err != nil {
	//		log.Error(err)
	//	}
	//}()

	r.ResponseOnce <- resp

	return nil
}

type CloseFn = func() error

type ReaderCloser struct {
	io.Reader
	Closes []CloseFn
}

func (c *ReaderCloser) Close() error {
	for i := range c.Closes {
		if err := c.Closes[i](); err != nil {
			return err
		}
	}
	return nil
}
