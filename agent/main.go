package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	log.SetFlags(0)

	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return errors.New("requires an address to listen on as the first argument")
	}

	l, err := net.Listen("tcp", os.Args[1])
	if err != nil {
		return err
	}

	log.Printf("agent is listening to commands from server on ws://%v", l.Addr())

	as := newAgentServer()
	httpServer := &http.Server{
		Handler:      as,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
	}

	errorChannel := make(chan error, 1)
	go func() {
		errorChannel <- httpServer.Serve(l)
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt)

	select {
	case err := <-errorChannel:
		log.Printf("failed to serve: %v", err)
	case sig := <-signalChannel:
		log.Printf("terminating: %v", sig)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	return httpServer.Shutdown(ctx)
}
