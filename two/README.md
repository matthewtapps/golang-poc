## Server
#### Running
```bash
go run . localhost:5567 # or desired port
```
```
```
Starts a http server listening for agents opening websockets on the provided address at the endpoing `/subscribe`.

After starting, allows commands to be passed to agents through the CLI (currently WIP).

## Agent
#### Running
```bash
go run . ws://127.0.0.1:5567 # or the address/port that the server was started on, prefixed with ws://
```

Opens a websocket to the server at the address specified, then awaits commands from that server.
```
```
