package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"

	"golang.org/x/time/rate"
)

type agentServer struct {
	logf func(f string, v ...interface{})
}

func (server agentServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	connection, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		server.logf("%v", err)
		return
	}

	defer connection.CloseNow()

	if connection.Subprotocol() != "echo" {
		connection.Close(websocket.StatusPolicyViolation, "client must speak the echo Subprotocol")
		return
	}

	limit := rate.NewLimiter(rate.Every(time.Millisecond*100), 10)
	for {
		err = agent(connection, limit)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			server.logf("failed to echo with %v: %v", request.RemoteAddr, err)
			return
		}
	}
}

func agent(connection *websocket.Conn, limit *rate.Limiter) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := limit.Wait(ctx)
	if err != nil {
		return err
	}

	_, reader, err := connection.Reader(ctx)
	if err != nil {
		return err
	}

	// Write received messages to stdout
	writer := io.Writer(os.Stdout)

	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to io.Copy: %w", err)
	}

	return err
}
