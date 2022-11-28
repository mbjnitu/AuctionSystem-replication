# Auction System

This repository is the answers to the assingment:
https://learnit.itu.dk/mod/assign/view.php?id=169980

## How to run

You have to run 3 servers, and an arbitrary amount of clients:

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
Within the clients the following 2 commands can be run:
1. "bid [amount]"
2. "result"

To simulate a server crash, simply click crtl+c in one of the 3 server terminals.
