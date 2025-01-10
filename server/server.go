package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"golang.org/x/time/rate"
)

type echoServer struct {
	logf func(f string, v ...interface{})
}

func (server echoServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
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
		err = echo(connection, limit)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return
		}
		if err != nil {
			server.logf("failed to echo with %v: %v", request.RemoteAddr, err)
		}
	}
}

func echo(connection *websocket.Conn, limit *rate.Limiter) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	err := limit.Wait(ctx)
	if err != nil {
		return err
	}

	messageType, reader, err := connection.Reader(ctx)
	if err != nil {
		return err
	}

	writer, err := connection.Writer(ctx, messageType)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, reader)
	if err != nil {
		return fmt.Errorf("failed to io.Copy: %w", err)
	}

	err = writer.Close()
	return err
}
