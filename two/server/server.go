package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/time/rate"
)

type authenticationServer struct {
	logf           func(f string, v ...interface{})
	serveMux       http.ServeMux
	agentsMu       sync.Mutex
	agents         map[*agent]struct{}
	commandLimiter *rate.Limiter
}

type agent struct {
	id        string
	commands  chan Command
	closeSlow func()
}

type Command struct {
	AgentId string
	Command string
	Ip      string
}

func newAuthServer() *authenticationServer {
	authServer := &authenticationServer{
		logf:           log.Printf,
		agents:         make(map[*agent]struct{}),
		commandLimiter: rate.NewLimiter(rate.Every(time.Millisecond*100), 8),
	}

	authServer.serveMux.HandleFunc("/subscribe", authServer.subscribeHandler)

	return authServer
}

func (as *authenticationServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.serveMux.ServeHTTP(w, r)
}

func (as *authenticationServer) subscribeHandler(w http.ResponseWriter, r *http.Request) {
	err := as.subscribe(w, r)
	if errors.Is(err, context.Canceled) {
		return
	}

	if websocket.CloseStatus(err) == websocket.StatusNormalClosure || websocket.CloseStatus(err) == websocket.StatusGoingAway {
		return
	}

	if err != nil {
		as.logf("%v", err)
		return
	}
}

func (as *authenticationServer) subscribe(w http.ResponseWriter, r *http.Request) error {
	var mu sync.Mutex
	var c *websocket.Conn
	var closed bool

	s := &agent{
		commands: make(chan Command, 16),
		id:       as.generateAgentId(),
		closeSlow: func() {
			mu.Lock()
			defer mu.Unlock()
			closed = true
			if c != nil {
				c.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with commands")
			}
		},
	}
	as.addAgent(s)
	defer as.deleteAgent(s)

	c2, err := websocket.Accept(w, r, nil)
	if err != nil {
		return err
	}
	mu.Lock()
	if closed {
		mu.Unlock()
		return net.ErrClosed
	}
	c = c2
	mu.Unlock()
	defer c.CloseNow()
	as.logf("accepted connection from agent, assigned id: %v", s.id)
	fmt.Print("-> ")

	ctx := c.CloseRead(context.Background())

	for {
		select {
		case command := <-s.commands:
			err := processCommand(ctx, time.Second*5, c, command)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			as.logf("connection to agent with id %v closed", s.id)
			return ctx.Err()
		}
	}
}

func processCommand(ctx context.Context, timeout time.Duration, c *websocket.Conn, command Command) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	marshalledMessage, err := json.Marshal(command)
	if err != nil {
		return err
	}

	// Writes the action as a stream of bytes to the websocket to the Agent
	return c.Write(ctx, websocket.MessageText, marshalledMessage)
}

func (as *authenticationServer) command(cmd []byte) {
	as.agentsMu.Lock()
	defer as.agentsMu.Unlock()

	as.commandLimiter.Wait(context.Background())

	command, err := decodeCommand(cmd)
	if err == errors.ErrUnsupported {
		log.Println("incorrect command format, must follow: <agent ID> <ip address> <...command>")
		fmt.Print("-> ")
		return
	} else if err != nil {
		as.logf("%v", err)
		return
	}

	var idRegistered bool

	for a := range as.agents {
		if command.AgentId == a.id {
			idRegistered = true
		}
	}

	if len(as.agents) < 1 || !idRegistered {
		as.logf("no agents registered for given id, doing nothing")
		fmt.Print("-> ")
		return
	}

	for a := range as.agents {
		if command.AgentId == a.id {
			select {
			case a.commands <- command:
			default:
				go a.closeSlow()
			}
		}
	}
}

func (as *authenticationServer) addAgent(a *agent) {
	as.agentsMu.Lock()
	as.agents[a] = struct{}{}
	as.agentsMu.Unlock()
}

func (as *authenticationServer) deleteAgent(a *agent) {
	as.agentsMu.Lock()
	delete(as.agents, a)
	as.agentsMu.Unlock()
}

func decodeCommand(cmd []byte) (Command, error) {
	stringCommand := string(cmd)
	splitString := strings.Split(stringCommand, " ")
	if len(splitString) < 3 {
		return Command{}, errors.ErrUnsupported
	}
	agentId := splitString[0]
	ipAddr := splitString[1]
	command := strings.Join(splitString[2:], " ")

	return Command{
		AgentId: agentId,
		Ip:      ipAddr,
		Command: command,
	}, nil
}

func (as *authenticationServer) generateAgentId() string {
	potentialId := 1
	for a := range as.agents {
		existingId, _ := strconv.Atoi(a.id)

		if existingId == potentialId {
			potentialId += 1
		}
	}
	return fmt.Sprintf("%v", potentialId)
}
