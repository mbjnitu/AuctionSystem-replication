package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	gRPC "github.com/PatrickMatthiesen/ChittyChat/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Same principle as in client. Flags allows for user specific arguments/values
var clientsName = flag.String("name", "default", "Senders name")
var serverPort = flag.String("server", "5400", "Tcp server")
var lamportTime = flag.Int64("lamport", 0, "Lamport time")

var server gRPC.TemplateClient  //the server
var ServerConn *grpc.ClientConn //the server connection

func main() {
	//parse flag/arguments
	flag.Parse()

	fmt.Println("--- CLIENT APP ---")

	//log to file instead of console
	f := setLog()
	defer f.Close()

	//connect to server and close the connection when program closes
	fmt.Println("--- join Server ---")
	ConnectToServer()
	go joinChat()
	defer ServerConn.Close()

	//start the biding
	parseInput()
}

// connect to server
func ConnectToServer() {
	//dial options
	//the server is not using TLS, so we use insecure credentials
	//(should be fine for local testing but not in the real world)
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	//dial the server, with the flag "server", to get a connection to it
	fmt.Printf("client %s: Attempts to dial on port %s\n", *clientsName, *serverPort)
	conn, err := grpc.Dial(fmt.Sprintf(":%s", *serverPort), opts...)
	if err != nil {
		fmt.Printf("Fail to Dial : %v", err)
		return
	}

	// makes a client from the server connection and saves the connection
	// and prints rather or not the connection was is READY
	server = gRPC.NewTemplateClient(conn)
	ServerConn = conn
	fmt.Println("the connection is: ", conn.GetState().String())
}

func joinChat() {
	joinRequest := &gRPC.JoinRequest{
		Name:        *clientsName,
		LamportTime: 0,
	}
	log.Println(*clientsName, " is joining the chat")
	stream, _ := server.Join(context.Background(), joinRequest)

	for {
		select {
		case <-stream.Context().Done():
			fmt.Println("Connection to server closed")
			return // stream is done
		default:
		}

		incomeing, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Server closed the stream")
			return
		}
		if err != nil {
			log.Fatalf("Failed to receive message from channel joining. \nErr: %v", err)
		}

		if incomeing.LamportTime > *lamportTime {
			*lamportTime = incomeing.LamportTime + 1
		} else {
			*lamportTime++
		}
		fmt.Printf("\rLamport: %v | %v: %v \n", *lamportTime, incomeing.Sender, incomeing.Message)
		fmt.Print("-> ")
	}
}

func parseInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Type the amount you wish to increment with here. Type 0 to get the current value")
	fmt.Println("--------------------")

	//Infinite loop to listen for clients input.
	fmt.Print("-> ")
	for {
		//Read input into var input and any errors into err
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input) //Trim input

		if !conReady(server) {
			log.Printf("Client %s: something was wrong with the connection to the server :(", *clientsName)
			continue
		}

		response, err := server.Publish(context.Background(), &gRPC.Message{
			Sender:      *clientsName,
			Message:     input,
			LamportTime: *lamportTime,
		})

		if err != nil {
			log.Printf("Client %s: something went wrong with the server :(", *clientsName)
			continue
		}
		if response.Message != "send" {
			log.Printf("Client %s: something went wrong with the server :(", *clientsName)
			continue
		}
	}
}

// Function which returns a true boolean if the connection to the server is ready, and false if it's not.
func conReady(s gRPC.TemplateClient) bool {
	return ServerConn.GetState().String() == "READY"
}

// sets the logger to use a log.txt file instead of the console
func setLog() *os.File {
	f, err := os.OpenFile("log.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	log.SetOutput(f)
	return f
}
