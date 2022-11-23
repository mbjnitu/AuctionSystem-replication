# Chitty Chat

For this assignment you needed to make a chat service.

The service is defined in [ChittyChat.proto](proto/ChittyChat.proto)

A user joins using the gRPC 'join' method.

A user publishes and broadcasts using the gRPC 'publish' method.

A user indicates to the server that it leaves when it stops the program.

## How to run

Server:

```sh
go run .\server\ 0
go run .\server\ 1
go run .\server\ 2
```

Client:

```sh
go run .\client\ -name alice
```
