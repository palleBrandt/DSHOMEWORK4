package node

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	TRS "p2p.com/proto"
)


//Creates the Node Struct. Definerer ligesom atts
type Node struct {
	wanted				bool
	Addr 				string
	Id 					int32
	mu 					sync.Mutex
	Peers 				map[string]TRS.TokenRingServiceClient
	VZ					TRS.VerbotenZoneServiceClient
	nextNode			string

	TRS.UnimplementedTokenRingServiceServer //Denne her er nødvendig for at Node implementerer server interfacet genereret af protofilen.
}

//Denne metode "åbner" nodens server funtionalitet op. Definerer hvilket port noden er på.
//Den springer ud af skabet som server.
func (node *Node) StartListening() {
	lis, err := net.Listen("tcp", node.Addr) //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	TRS.RegisterTokenRingServiceServer(grpcServer, node) //Dette registrerer noden som en værende en TokenRingServiceServer.

	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

//Dette er nodens "main" function.
func (node *Node) Start() error {
	node.ConnectToVZ()
	node.wanted = false;
	node.Peers = make(map[string]TRS.TokenRingServiceClient) //Instantierer nodens map over peers.

	//Hardcoded list af servere
	hardcodedIPs := []string{"localhost:50051", "localhost:50052", "localhost:50053"}
	go node.StartListening() //Go routine med kald til "server" funktionaliteten.

	foundMatchingAddr := false
	neverfoundmatch := true
	// Run through each of the IPs
	for _, addr := range hardcodedIPs {
		// Skip the current node
		if addr == node.Addr {
			foundMatchingAddr = true
			neverfoundmatch = false
			continue // If the addr is the addr of the node, skip this.
		}

		// Check if the matching address is found
		if foundMatchingAddr {
			// Perform your desired action with the addr here
			node.nextNode = addr
			foundMatchingAddr = false	
		}

		// Setup connection to each other node, and map the connection to the addr in peers.
		node.SetupClient(addr)
	}
	if neverfoundmatch {
		node.nextNode = hardcodedIPs[0]
	}

	//If you are the first node, then you are the first with token, and you start the ring.
	if node.Id == 1{
		node.Peers[node.nextNode].GiveToken(context.Background(), &TRS.Token{Token: true})
	}
	for {
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
		node.mu.Lock()
		node.wanted = true
		node.mu.Unlock()

		for {
			if node.wanted == false {
				break
			}
		}

	}
}

func (node *Node) ConnectToVZ() {
	conn, err := grpc.Dial("localhost:50054", grpc.WithInsecure()) //Dial op connection to the address
	if err != nil {
		log.Printf("Unable to connect to VZ")
		return
	}
	node.mu.Lock()
	node.VZ = TRS.NewVerbotenZoneServiceClient(conn)
	node.mu.Unlock()
}

//Denne metode skaber forbindelsen til de andre noder, som for denne ene node forståes som clienter.
func (node *Node) SetupClient(addr string) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure()) //Dial op connection to the address
	if err != nil {
		log.Printf("Unable to connect to %s: %v", addr, err)
		return
	}

	node.mu.Lock()
	node.Peers[addr] = TRS.NewTokenRingServiceClient(conn) //Create a new tokenringclient and map it with its address.
	fmt.Println("Node(", node.Addr ,") has connected to Node(",addr, ")") //Print that the connection happened.
	node.mu.Unlock()
}

//Grpc endpoint.
func (node *Node) GiveToken(ctx context.Context, tok *TRS.Token) (*TRS.Ack, error) {
	if node.wanted {
		//access verboten zone
		_, err := node.VZ.GoIn(context.Background(), &TRS.VerbotenZoneMsg{Id: node.Id}) //Dial op connection to the address
		if err != nil {
			log.Printf("Unable to access verboten zone")
			return &TRS.Ack{Status: 401}, err
		}
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)

		//leave verbotenzone
		response, err := node.VZ.GoOut(context.Background(), &TRS.VerbotenZoneMsg{Id: node.Id}) //Dial op connection to the address
		if err != nil {
			log.Printf("unable:", response.Status)
			return &TRS.Ack{Status: 401}, err
		}
		node.mu.Lock()
		node.wanted = false
		node.mu.Unlock()
	}

	//pass token
	nextNodeClient := node.Peers[node.nextNode]
	_, err := nextNodeClient.GiveToken(context.Background(), &TRS.Token{Token: true}) //Dial op connection to the address
	if err != nil {
		log.Printf("Unable to pass token")
		return &TRS.Ack{Status: 401}, err
	}
	return &TRS.Ack{Status: 200}, err
}
