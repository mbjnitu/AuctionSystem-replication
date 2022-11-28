package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	// this has to be the same as the go.mod module,
	// followed by the path to the folder the proto file is in.
	gRPC "github.com/mbjnitu/AuctionSystem-replication/proto"

	"google.golang.org/grpc"
)

type Server struct {
	gRPC.UnimplementedAuctionSystemServer        // You need this line if you have a server struct
	port                                  string // Not required but useful if your server needs to know what port it's listening to

	streams map[string]*gRPC.AuctionSystem_JoinServer // map of streams
}

// flags are used to get arguments from the terminal. Flags take a value, a default value and a description of the flag.
// to use a flag then just add it as an argument when running the program.
var port *string        // set with "-port <port>" in terminal
var serverName = "port" // name of the server
var serverPort string   // port of the server port
var serverSomething = "server port"

// Vars related to bidding:
var currentAmount int64 = 0
var auctionOver bool = false

func main() {
	arg1, _ := strconv.ParseInt(os.Args[1], 10, 64)
	serverPort32 := int64(arg1) + 5000
	serverPort = strconv.FormatInt(serverPort32, 10)
	fmt.Println(serverPort)
	port = flag.String(serverName, serverPort, serverSomething)
	f := setLog() //uncomment this line to log to a log.txt file instead of the console
	defer f.Close()

	// This parses the flags and sets the correct/given corresponding values.
	flag.Parse()
	fmt.Println(".:server is starting:.")

	// starts a goroutine executing the launchServer method.
	launchServer()

	go endAuction()

	// code here is unreachable because launchServer occupies the current thread.
}

func launchServer() {
	fmt.Printf("Attempts to create listener on port %s\n", *port)

	// Create listener tcp on given port or default port 5400
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%s", *port))
	if err != nil {
		fmt.Printf("Failed to listen on port %s: %v", *port, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}

	// makes gRPC server using the options
	// you can add options here if you want or remove the options part entirely
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	// makes a new server instance using the name and port from the flags.
	server := &Server{
		port:    *port,
		streams: make(map[string]*gRPC.AuctionSystem_JoinServer),
	}

	gRPC.RegisterAuctionSystemServer(grpcServer, server) //Registers the server to the gRPC server.

	fmt.Println("Clients should dial: ", GetOutboundIP())

	if err := grpcServer.Serve(list); err != nil {
		log.Fatalf("failed to serve %v", err)
	}
	// code here is unreachable because grpcServer.Serve occupies the current thread.
}

func (s *Server) Join(request *gRPC.JoinRequest, stream gRPC.AuctionSystem_JoinServer) error {
	log.Printf("Server: Join request from %s\n", request.Name)

	// adds the stream to the streams map
	s.streams[request.Name] = &stream

	// sends a message to the client
	sendToAll(s.streams, &gRPC.Message{
		Sender:  "Server",
		Message: "Welcome " + request.Name + " to the auction!",
		Bid:     0,
	})

	// waits for the stream to be closed -- happens when the client stops
	// then removes the stream from the streams map
	// and sends a message to the other clients
	<-stream.Context().Done()
	delete(s.streams, request.Name)
	log.Println(request.Name, "disconnected")

	sendToAll(s.streams, &gRPC.Message{
		Sender:  "Server",
		Message: request.Name + " has left the auction",
		Bid:     0,
	})
	return nil
}

func (s *Server) Publish(ctx context.Context, message *gRPC.Message) (*gRPC.PublishResponse, error) {

	println("omg i received a message: ", message.Message)

	processInput(message, s.streams)

	return &gRPC.PublishResponse{}, nil
}

func processInput(message *gRPC.Message, streams map[string]*gRPC.AuctionSystem_JoinServer) {
	if message.Message == "bid" && !auctionOver {
		if message.Bid > currentAmount {
			currentAmount = message.Bid
			sendToAll(streams, &gRPC.Message{
				Sender:  "Server",
				Message: "A new highest bet has been set by " + message.Sender + "with a value of: ",
				Bid:     currentAmount,
			})
		} else if message.Bid <= currentAmount {
			sendToSpecific(streams, &gRPC.Message{
				Sender:  "Server",
				Message: "Your bid is not greater than the current highest bid of: ",
				Bid:     currentAmount,
			}, message.Sender)
		}
	} else if message.Message == "result" && !auctionOver {
		sendToSpecific(streams, &gRPC.Message{
			Sender:  "Server",
			Message: "The current result is: ",
			Bid:     currentAmount,
		}, message.Sender)
	} else if auctionOver {
		sendToAll(streams, &gRPC.Message{
			Sender:  "Server",
			Message: "The auction is over, and was win by " + message.Sender + "at the price: ",
			Bid:     currentAmount,
		})
	}
}

// sends a message to a specific stream in the streams map
func sendToSpecific(streams map[string]*gRPC.AuctionSystem_JoinServer, message *gRPC.Message, sender string) {
	stream := streams[sender]
	(*stream).Send(message)
	fmt.Println("server")
}

// sends a message to all streams in the streams map
func sendToAll(streams map[string]*gRPC.AuctionSystem_JoinServer, message *gRPC.Message) {
	for _, stream := range streams {
		(*stream).Send(message)
	}
	fmt.Println("server")
}

// Get preferred outbound ip of this machine
// Usefull if you have to know which ip you should dial, in a client running on an other computer
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// sets the logger to use a log.txt file instead of the console
func setLog() *os.File {
	// Clears the log.txt file when a new server is started
	if err := os.Truncate("log.txt", 0); err != nil {
		log.Printf("Failed to truncate: %v", err)
	}

	// This connects to the log file/changes the output of the log informaiton to the log.txt file.
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}

func endAuction() {
	time.Sleep(120 * time.Second)
	auctionOver = true
	log.Printf("Auction has ended")
}
