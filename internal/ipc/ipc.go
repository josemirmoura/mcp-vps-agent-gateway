package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type Handler interface {
	Handle(context.Context, wire.Request) wire.Response
}

type Server struct {
	socket  string
	handler Handler
	ln      net.Listener
	wg      sync.WaitGroup
}

func NewServer(socket string, handler Handler) *Server {
	return &Server{socket: socket, handler: handler}
}

func (s *Server) Serve(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.socket), 0o750); err != nil {
		return err
	}
	_ = os.Remove(s.socket)
	ln, err := net.Listen("unix", s.socket)
	if err != nil {
		return err
	}
	s.ln = ln
	if err := os.Chmod(s.socket, 0o660); err != nil {
		ln.Close()
		return err
	}

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				break
			}
			return err
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer conn.Close()
			var req wire.Request
			if err := json.NewDecoder(conn).Decode(&req); err != nil {
				_ = json.NewEncoder(conn).Encode(wire.ErrorResponse("", "bad_request", err.Error()))
				return
			}
			resp := s.handler.Handle(ctx, req)
			_ = json.NewEncoder(conn).Encode(resp)
		}()
	}
	s.wg.Wait()
	return nil
}

type Client struct {
	Socket  string
	Timeout time.Duration
}

func (c Client) Call(ctx context.Context, req wire.Request) (wire.Response, error) {
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	d := net.Dialer{Timeout: timeout}
	conn, err := d.DialContext(ctx, "unix", c.Socket)
	if err != nil {
		return wire.Response{}, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return wire.Response{}, err
	}
	var resp wire.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return wire.Response{}, err
	}
	return resp, nil
}
