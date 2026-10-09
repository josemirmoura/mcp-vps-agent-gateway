package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/josemirmoura/mcp-vps-agent-gateway/internal/wire"
)

type Handler interface {
	Handle(context.Context, wire.Request) wire.Response
}

type Server struct {
	socket          string
	handler         Handler
	ln              net.Listener
	wg              sync.WaitGroup
	AllowedPeerUIDs map[uint32]struct{}
	SocketGroupGID int // explicit group for a dedicated, minimal-privilege socket; 0 = preserve existing behavior
}

func NewServer(socket string, handler Handler) *Server {
	return &Server{socket: socket, handler: handler}
}

func (s *Server) AllowPeerUIDs(uids ...uint32) {
	if s.AllowedPeerUIDs == nil {
		s.AllowedPeerUIDs = make(map[uint32]struct{}, len(uids))
	}
	for _, uid := range uids {
		s.AllowedPeerUIDs[uid] = struct{}{}
	}
}

func unixPeerUID(conn net.Conn) (uint32, error) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, errors.New("peer is not a Unix connection")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *unix.Ucred
	var controlErr error
	if err := raw.Control(func(fd uintptr) {
		cred, controlErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil {
		return 0, err
	}
	if controlErr != nil {
		return 0, controlErr
	}
	if cred == nil {
		return 0, errors.New("missing Unix peer credentials")
	}
	return cred.Uid, nil
}

func (s *Server) peerAllowed(conn net.Conn) bool {
	if len(s.AllowedPeerUIDs) == 0 {
		return true
	}
	uid, err := unixPeerUID(conn)
	if err != nil {
		return false
	}
	_, ok := s.AllowedPeerUIDs[uid]
	return ok
}

func (s *Server) Serve(ctx context.Context) error {
	if err := os.MkdirAll(filepath.Dir(s.socket), 0o750); err != nil {
		return err
	}
	if s.SocketGroupGID > 0 {
		// This is a dedicated mount owned by the privileged Broker. The portal
		// gets only group traversal rights for its separate operator socket.
		if err := os.Chown(filepath.Dir(s.socket), -1, s.SocketGroupGID); err != nil { return err }
		if err := os.Chmod(filepath.Dir(s.socket), 0o750); err != nil { return err }
	}
	_ = os.Remove(s.socket)
	ln, err := net.Listen("unix", s.socket)
	if err != nil {
		return err
	}
	s.ln = ln
	// #nosec G302 -- 0660 is intentional: Broker owns the socket and the dedicated Gateway group requires read/write access.
	if err := os.Chmod(s.socket, 0o660); err != nil {
		_ = ln.Close()
		return err
	}

	if s.SocketGroupGID > 0 {
		if err := os.Chown(s.socket, -1, s.SocketGroupGID); err != nil {
			_ = ln.Close()
			return err
		}
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
		if !s.peerAllowed(conn) {
			_ = json.NewEncoder(conn).Encode(wire.ErrorResponse("", "unauthorized_peer", "Unix peer credential is not authorized"))
			_ = conn.Close()
			continue
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer conn.Close()
			var req wire.Request
			const maxRequestBytes = 2 << 20
			if err := json.NewDecoder(io.LimitReader(conn, maxRequestBytes+1)).Decode(&req); err != nil {
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
