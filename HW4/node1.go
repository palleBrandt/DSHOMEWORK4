package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	gRPC "mutual.com/proto"
)

var url string = "localhost:4201";

func main(){
	fmt.Println("hej");
	addresses := []string{
        "192.168.1.101:4201",
        "192.168.1.102:4202",
        "192.168.1.103:4203",
    }
	

	list, _ := net.Listen("tcp", url)
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)


	server := &Server{
		t: 0,
		requests: make ([]gRPC.ReqMes, 0, 200)}
		
	
	gRPC.RegisterRicartAgrawalaServer(grpcServer, server)
	grpcServer.Serve(list)
	
	ConnectToNodes();
}



type Node struct {
    // an interface that the server type needs to have
    gRPC.UnimplementedRicartAgrawalaServer

	// A list of all streams created between the clients and the server
	requests	[]*gRPC.ReqMes;
	t int32;
	state string;
	respondCounter int32;
	addresses []string;
	nodes []gRPC.RicartAgrawalaClient;
}

func (n *Node) RequestAcces(ctx context.Context, ReqMes *gRPC.ReqMes) (*gRPC.Ack, error) {


	if n.state == "HELD" || 
		(n.state == "WANTED" &&
			(n.t < ReqMes.LamportTimestamp)) {
				n.requests = append(n.requests, ReqMes);
				n.t = maxInt32(n.t, ReqMes.LamportTimestamp) + 1;
			} else {
				n.t = maxInt32(n.t, ReqMes.LamportTimestamp) + 1;
				//update max ts +1
				//reply

			}
}

func (n *Node) RespondAcces(ctx context.Context, ReqMes *gRPC.ReqMes) (*gRPC.Ack, error) {
	n.t = maxInt32(n.t, ReqMes.LamportTimestamp) + 1;

}

func reply {
	
}

//is called when a client joins the "chitty chat". This method is bidirectional streaming. On the client side the stream is saved 
// and used to send messages. The stream returned on the client side, is used to publish messages to the server, and to send an initial join message.
// This is the core of functionality in chitty chat.
func (n *Node) Subscribe (stream gRPC.ChittyChat_SubscribeServer) error {

	s.Lock();
	s.clients = append(s.clients, stream);
	s.Unlock();

	 clientMessage, err := stream.Recv() // Receive a an initial chat message sent by the client after subscription. This is used solely to identify the
	 //client in this stream.
        if err != nil {
            fmt.Println(err);
        }
	s.t = maxInt32(s.t, clientMessage.LamportTimestamp) + 1
	s.Join(clientMessage); //Join message is called with the client message. This is a method that handles the broadcasting of "somebody has joined"


	//This loop listens for incoming messages being published form the client.
	for {
        message, err := stream.Recv()
		//If the error is not null, we assume that the connection to the client is lost. AKA the client has disconnected
        if err != nil {
            s.Lock()
            for i, client := range s.clients {
                if client == stream {
					//Therefore the client is removed from the saved clients (streams) so we do not try to publish to it.
                    s.clients = append(s.clients[:i], s.clients[i+1:]...)//Append everything up til i, append everything after.
					s.Leave(clientMessage) //Leave method is called, to broadcast the "somebody left" message.
                    break
                }
            }
            s.Unlock()
            return err
        }
        s.Lock()
		//Increments timestamp for recieving a message
		s.t = maxInt32(s.t, message.LamportTimestamp) + 1;
        s.Unlock()
		//Increments timestamp for sending a message
		s.t ++;
		//Updates the messages timestamp
		message.LamportTimestamp = s.t;
        s.broadcast(message) // Broadcast the new message to all connected clients
    }
}

// Sends the message to all streams in the Cliens list.
func (s *Server) Join (message *gRPC.Message) error{
	//Increments timestamp for when a client joins the server
	s.t ++;
	joinMessage := &gRPC.Message{AuthorName: "server", Text: "Participant " + message.AuthorName + " joined Chitty-Chat at Lamport time: " + strconv.FormatInt(int64(s.t), 10), LamportTimestamp: s.t};
	s.broadcast(joinMessage);
	return nil
}

// Sends the message to all streams in the Cliens list.
func (s *Server) Leave (message *gRPC.Message) error{ 
	//Increments timestamp for when a client leaves the server
	s.t ++;
	leaveMessage := &gRPC.Message{AuthorName: "server", Text: "Participant " + message.AuthorName + " left Chitty-Chat at Lamport time: " + strconv.FormatInt(int64(s.t), 10), LamportTimestamp: s.t};
	s.broadcast(leaveMessage);
	return nil
}

// Sends the message to all streams in the Cliens list.
func (s *Server) broadcast (message *gRPC.Message) error{
	for _, client := range s.clients {
			if err := client.Send(message); err != nil {
				return err
			}
	}
	return nil
}

func maxInt32(a, b int32) int32 {
		if a > b {
			return a
		}
		return b
	}

func ConnectToNodes(){
	for i, address := range n.addresses {
		fmt.Println("Test")
                if address != url {
					var opts []grpc.DialOption
					opts = append(
						opts, grpc.WithBlock(), 
						grpc.WithTransportCredentials(insecure.NewCredentials()),	
					)
					
					conn, err := grpc.Dial(address, opts...)
					if err != nil {
						log.Printf("Fail to Dial : %v", err)
						return
					}
					
					node = gRPC.NewRicardAgrawalaClient(conn)
					nodes = append(nodes, node)
					log.Println("the connection is: ", conn.GetState().String())
                }
            }
}