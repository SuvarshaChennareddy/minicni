package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/suvarshachennareddy/minicni/common/cio"
	"github.com/suvarshachennareddy/minicni/common/proto"
	"github.com/suvarshachennareddy/minicni/daemon/internal/manager"
)

type Server struct {
	listener net.Listener
	manager  manager.Manager
}

func New(sockPath string, m manager.Manager) (*Server, error) {
	l, err := net.Listen("unix", sockPath)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener: l,
		manager:  m,
	}, nil
}

func (s *Server) Serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.listener.Close()
	}()

	log.Println("Server listening")

	for {
		c, err := s.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			log.Fatalf("failed to accept connection: %v", err)

			return
		}

		go s.handleConn(ctx, c)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.Close()

	var err error
	cniArgs := &proto.CNIArgs{}

	_, err = cio.ReadObj(conn, cniArgs)
	if err != nil {
		log.Printf("Failed to read from socket: %v", err)

		return
	}

	statusResponse := &proto.StatusResponse{}

	handleErr := s.manager.Handle(ctx, cniArgs)
	if handleErr != nil {
		statusResponse.Success = false
		statusResponse.Error = handleErr.Error()

		log.Printf("Handle failed with args %+v: %v", cniArgs, handleErr)
	} else {
		statusResponse.Success = true
	}

	_, err = cio.WriteObj(conn, statusResponse)
	if err != nil {
		log.Printf("Failed to write to socket: %v", err)

		return
	}

	fmt.Printf("Recieved: %+v\n", cniArgs)
}
