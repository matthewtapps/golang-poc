package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

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
		return errors.New("must specify server address and port as first argument")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connection, _, err := websocket.Dial(ctx, fmt.Sprintf("%s%s", os.Args[1], "/subscribe"), &websocket.DialOptions{
		Subprotocols: []string{"echo"},
	})
	if err != nil {
		return err
	}

	log.Println("websocket to server established, awaiting commands")
	defer connection.Close(websocket.StatusInternalError, "connection to server closed")

	for {
		err := receiveMessage(ctx, connection)
		if err != nil {
			return err
		}
	}
}

type AuthServerCommand struct {
	Command Command
	Ip      string
}

type Command string

const (
	AllowIpAddr Command = "AllowIpAddr"
	BlockIpAddr Command = "BlockIpAddr"
)

func receiveMessage(ctx context.Context, connection *websocket.Conn) error {
	_, reader, err := connection.Reader(ctx)
	if err != nil {
		return err
	}

	message, err := io.ReadAll(reader)

	var v AuthServerCommand

	err = json.Unmarshal(message, &v)
	if err != nil {
		return err
	}

	switch v.Command {
	case AllowIpAddr:
		log.Printf("Command received to allow IP Address %v", v.Ip)
	case BlockIpAddr:
		log.Printf("Command received to block IP Address %v", v.Ip)
	default:
		log.Printf("Ip: %v, Command: %v", v.Ip, v.Command)
	}

	return nil
}
