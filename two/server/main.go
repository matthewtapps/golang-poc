package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

func main() {
	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		log.Fatalf("error setting up OTeL SDK - %e", err)
	}
	defer otelShutdown()

	log.SetFlags(0)
	err = run()
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

	log.Printf("server is waiting for agents to establish connections on ws://%v", l.Addr())

	as := newAuthServer()
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

	commandChannel := make(chan []byte, 3)
	go scan(commandChannel)

	for {
		select {
		case err := <-errorChannel:
			log.Printf("failed to serve: %v", err)
			return err
		case sig := <-signalChannel:
			log.Printf("terminating: %v", sig)
			return nil
		case cmd, ok := <-commandChannel:
			if !ok {
				as.logf("command channel closed")
				return nil
			}
			go as.command(cmd)
		}
	}
}

func scan(cmdChan chan<- []byte) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("-> ")
	for {
		if scanner.Scan() {
			cmdChan <- scanner.Bytes()
			fmt.Print("-> ")
		} else {
			log.Printf("scanner stopped or input closed")
			close(cmdChan)
			return
		}
	}
}
