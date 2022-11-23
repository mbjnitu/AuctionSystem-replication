package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	gRPC "github.com/PatrickMatthiesen/ChittyChat/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Same principle as in client. Flags allows for user specific arguments/values
var clientsName = flag.String("name", "default", "Senders name")
var serverPort0 = flag.String("server0", "5000", "Tcp server")
var serverPort1 = flag.String("server1", "5001", "Tcp server")
var serverPort2 = flag.String("server2", "5001", "Tcp server")
var lamportTime = flag.Int64("lamport", 0, "Lamport time")

var servers = []gRPC.ChittyChatClient{}
var serverConns []*grpc.ClientConn

func main() {

	//parse flag/arguments
	flag.Parse()

	fmt.Println("--- Welcome to Chitty Chat ---")

	//log to file instead of console
	f := setLog()
	defer f.Close()

	//connect to server and close the connection when program closes
	for i := 0; i < 3; i++ {
		serverName := "server" + strconv.Itoa(i) // name of the server
		serverPort := "500" + strconv.Itoa(i)    // port of the server port
		fmt.Println("Servername: " + serverName + " - Serverport: " + serverPort)
		connectToServer(serverName, serverPort)
	}
	go joinChat()

	// start allowing user input
	for i := 0; i < 3; i++ {
		parseAndSendInput(i)
	}
}

// connect to server
func connectToServer(serverName string, serverPort string) {
	serverFlag := serverPort
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))

	fmt.Printf("client %s: Attempts to dial on port %s\n", *clientsName, serverFlag)
	conn, err := grpc.Dial(fmt.Sprintf(":%s", serverFlag), opts...)
	if err != nil {
		fmt.Printf("Fail to Dial : %v", err)
		return
	}

	servers = append(servers, gRPC.NewChittyChatClient(conn))
	serverConns = append(serverConns, conn)
}

func joinChat() {
	joinRequest := &gRPC.JoinRequest{
		Name:        *clientsName,
		LamportTime: 0,
	}
	log.Println(*clientsName, "is joining the chat")
	for i := 0; i < 3; i++ {
		stream, _ := servers[i].Join(context.Background(), joinRequest)
		go awaitResponse(stream)
	}
}

func awaitResponse(stream gRPC.ChittyChat_JoinClient) {
	for {
		select {
		case <-stream.Context().Done():
			fmt.Println("Connection to server closed")
			return // stream is done
		default:
		}

		incoming, err := stream.Recv()
		if err == io.EOF {
			fmt.Println("Server is done sending messages")
			return
		}
		if err != nil {
			log.Fatalf("Failed to receive message from channel. \nErr: %v", err)
		}

		if incoming.LamportTime > *lamportTime {
			*lamportTime = incoming.LamportTime + 1
		} else {
			*lamportTime++
		}

		log.Printf("%s got message from %s: %s", *clientsName, incoming.Sender, incoming.Message)
		fmt.Printf("\rLamport: %v | %v: %v \n", *lamportTime, incoming.Sender, incoming.Message)
		fmt.Print("-> ")
	}
}

func parseAndSendInput(serverNumber int) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Type the amount you wish to increment with here. Type 0 to get the current value")
	fmt.Println("--------------------")

	//Infinite loop to listen for clients input.
	fmt.Print("-> ")
	for {
		//Read user input to the next newline
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input) //Trim whitespace

		// we are sending a message so we increment the lamport time
		*lamportTime++

		// publish the message in the chat
		response, err := servers[serverNumber].Publish(context.Background(), &gRPC.Message{
			Sender:      *clientsName,
			Message:     input,
			LamportTime: *lamportTime,
		})

		if err != nil || response == nil {
			log.Printf("Client %s: something went wrong with the server :(", *clientsName)
			continue
		}
	}
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
