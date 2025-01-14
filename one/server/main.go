package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coder/websocket"
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

	connection, _, err := websocket.Dial(ctx, fmt.Sprintf("%s%s", os.Args[1], "/subscribe"), &websocket.DialOptions{
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
			writer, err := connection.Writer(ctx, websocket.MessageText)
			if err != nil {
				return err
			}

			v := map[string]string{
				"Message": scanner.Text(),
			}

			marshalledMessage, err := json.Marshal(v)
			if err != nil {
				return err
			}

			writer.Write(marshalledMessage)
			writer.Close()
			fmt.Print("-> ")
		}
	}
}
