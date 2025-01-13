package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"

	"golang.org/x/time/rate"
)

type agentServer struct {
	logf           func(f string, v ...interface{})
	publishLimiter *rate.Limiter
	serveMux       http.ServeMux
	subscribersMu  sync.Mutex
	messages       chan []byte
}

func newAgentServer() *agentServer {
	agentServer := &agentServer{
		logf:           log.Printf,
		publishLimiter: rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
	}

	agentServer.serveMux.HandleFunc("/subscribe", agentServer.subscribeHandler)

	return agentServer
}

func (as *agentServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.serveMux.ServeHTTP(w, r)
}

func (as *agentServer) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	err := as.subscribe(w, r)
	if errors.Is(err, context.Canceled) {
		return
	}
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
		websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		as.logf("%v", err)
		return
	}
}

func (as *agentServer) subscribe(writer http.ResponseWriter, request *http.Request) error {
	connection, err := websocket.Accept(writer, request, nil)
	if err != nil {
		return err
	}

	println("accepted connection from server")
	defer connection.CloseNow()

	for {
		err = as.decode(connection)
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			return err
		}
		if err != nil {
			as.logf("agent failed to action message with %v: %v", request.RemoteAddr, err)
			return err
		}
	}
}

func (as *agentServer) decode(connection *websocket.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()
	writer := io.Writer(os.Stdout)

	err := as.publishLimiter.Wait(ctx)
	if err != nil {
		return err
	}

	_, reader, err := connection.Reader(ctx)
	if err != nil {
		return err
	}

	message, err := io.ReadAll(reader)

	type Message struct {
		Message string
	}
	var v Message

	err = json.Unmarshal(message, &v)
	if err != nil {
		return err
	}

	for {
		writer.Write([]byte(v.Message))
		writer.Write([]byte("\n"))
		return nil
	}
}
