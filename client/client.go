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

	gRPC "github.com/mbjnitu/AuctionSystem-replication/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Same principle as in client. Flags allows for user specific arguments/values
var clientsName = flag.String("name", "default", "Senders name")
var serverPort0 = flag.String("server0", "5000", "Tcp server")
var serverPort1 = flag.String("server1", "5001", "Tcp server")
var serverPort2 = flag.String("server2", "5001", "Tcp server")

var servers = []gRPC.AuctionSystemClient{}
var serverConns []*grpc.ClientConn

func main() {

	//parse flag/arguments
	flag.Parse()

	fmt.Println("--- Welcome to the auction---")

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
	parseAndSendInput()
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

	servers = append(servers, gRPC.NewAuctionSystemClient(conn))
	serverConns = append(serverConns, conn)
}

func joinChat() {
	joinRequest := &gRPC.JoinRequest{
		Name: *clientsName,
	}
	log.Println(*clientsName, "is joining the auction")
	for i := 0; i < 3; i++ {
		stream, _ := servers[i].Join(context.Background(), joinRequest)
		awaitResponse(stream)
	}
}

func awaitResponse(stream gRPC.AuctionSystem_JoinClient) {
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

		fmt.Println(incoming.Message + strconv.FormatInt(incoming.Bid, 10))
		log.Printf(incoming.Message + strconv.FormatInt(incoming.Bid, 10) + "\n")
	}
}

func parseAndSendInput() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Type \"bid [amount]\" or \"result\" to interact with the auction system")
	fmt.Println("--------------------")

	//Infinite loop to listen for clients input.
	for {
		//Read user input to the next newline
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		input = strings.TrimSpace(input) //Trim whitespace

		processInput(input)
	}
}

// Determines if input is valid, and if its a "bid" or "result" request.
func processInput(input string) {
	if strings.Contains(input, "bid ") {
		bid, _ := strconv.ParseInt(strings.Split(input, " ")[1], 10, 64)

		for i := 0; i < 3; i++ {
			response, err := servers[i].Publish(context.Background(), &gRPC.Message{
				Sender:  *clientsName,
				Message: "bid",
				Bid:     bid,
			})
			if err != nil || response == nil {
				log.Printf("Client %s: something went wrong with the server :(", *clientsName)
				continue
			}
		}
	} else if strings.Contains(input, "result") {
		for i := 0; i < 3; i++ {
			response, err := servers[i].Publish(context.Background(), &gRPC.Message{
				Sender:  *clientsName,
				Message: "result",
				Bid:     -1,
			})
			if err != nil || response == nil {
				log.Printf("Client %s: something went wrong with the server :(", *clientsName)
				continue
			}
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
