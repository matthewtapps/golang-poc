## Server
#### Running
```bash
go run . localhost:5567 # or desired port
```
Starts a http server listening for agents opening websockets on the provided address at the endpoing `/subscribe`.

After starting, allows commands to be passed to agents through the CLI.

Agents are assigned incremental IDs as they connect. Commands can be passed to specific agents via their ID. Only one agent can be sent a command at a time.

#### Using
```bash
<agent ID> <ip address (currently any string will be accepted)> <...command (any number of strings)>
1 192.168.0.4 test command
```
## Agent
#### Running
```bash
go run . ws://127.0.0.1:5567 # or the address/port that the server was started on, prefixed with ws://
```
Opens a websocket to the server at the address specified, then awaits commands from that server.

When commands are received, echoes the IP address and command that was sent to the console output.
