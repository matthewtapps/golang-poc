package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
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
		return errors.New("must specify the address that the agent is listening on as the first argument")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	connection, _, err := websocket.Dial(ctx, os.Args[1], &websocket.DialOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		return err
	}
	defer connection.Close(websocket.StatusInternalError, "connection to agent closed")

	fmt.Println("Enter message for agent and submit")

	for {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("-> ")
		for scanner.Scan() {
			err = wsjson.Write(ctx, connection, map[string]string{
				"type":    "echo",
				"message": scanner.Text(),
			})
			if err != nil {
				return err
			}
			fmt.Print("-> ")
		}
	}
}
