package node

import (
	"fmt"
	"log"
	"net"
	"math/rand"
	"time"
	"sync"
	
	"golang.org/x/net/context"
	ras "p2p.com/proto"
	"google.golang.org/grpc"
)

//De forskellige states en node kan have.
type State int

const (
	RELEASED State = iota //Gør at værdierne forståes som integers, og sætter RELEASED til 0, WANTED til 1 og HELD til 2. Samt sætter RELEASED til initial værdi af state.
	WANTED
	HELD
)

//Creates the Node Struct. Definerer ligesom atts
type Node struct {

	state 				State
	Addr 				string
	Id 					int
	ResponseCounter 	int
	LPTimestamp 		int
	mu 					sync.Mutex
	Peers 				map[string]ras.RicartAgrawalaClient
	ReqQueue			[]string

	ras.UnimplementedRicartAgrawalaServer //Denne her er nødvendig for at Node implementerer server interfacet genereret af protofilen.
}

//Denne metode "åbner" nodens server funtionalitet op. Definerer hvilket port noden er på.
//Den springer ud af skabet som server.
func (node *Node) StartListening() {
	lis, err := net.Listen("tcp", node.Addr) //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	ras.RegisterRicartAgrawalaServer(grpcServer, node) //Dette registrerer noden som en værende en HelloServiceServer.

	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

//Dette er nodens "main" function.
func (node *Node) Start() error {
	node.Peers = make(map[string]ras.RicartAgrawalaClient) //Instantierer nodens map over peers.
	node.LPTimestamp = 0
	node.ResponseCounter = 0
	node.State = State.RELEASED
	node.ReqQueue = make([]string, 0 , 10)
	go node.StartListening() //Go routine med kald til "server" funktionaliteten.

	//Hardcoded list af servere
	hardcodedIPs := []string{"localhost:50051", "localhost:50052", "localhost:50053"}

	//Run through each of the IPs
	for _, addr := range hardcodedIPs {
		// Skip the current node
		if addr == node.Addr {
			continue //If the addr is the addr of the node, skip this.
		}

		// Setup connection to each other node, and map the connection to the addr in peers.
		node.SetupClient(addr)
	}

	//Dette er mest bare en default funktionalitet jeg har lagt ind i noderne for at man kan se at der sker noget og at forbindelserne virker.
	//Hver node sover mellem 0-5 sekunder, og så kalder de ellers sayHello gRPC endpoint i sine peers.
	for {
		time.Sleep(time.Duration(rand.Intn(5)) * time.Second)
		// for key, peer := range node.Peers {
		// 	//Den giver sin egen adresse med i kaldet.
		// 	fmt.Println("I am Node(", node.Id, ") and i am calling Request on Node(",  key, ")")
		// 	fmt.Println("...")
		// 	response, err := peer.Request(context.Background(), &ras.HelloRequest{Name: node.Addr})
    	// 	if err != nil {
        // 		fmt.Println("Error making request to %s: %v", peer, err)
    	// 	}
		// 	//Og her printer den ellers respons message.
		// 	fmt.Println("I am Node(", node.Addr, ") and i got this response", response.Message,)
		// 	fmt.Println("...")
		// }
	}
}

//Denne metode skaber forbindelsen til de andre noder, som for denne ene node forståes som clienter.
func (node *Node) SetupClient(addr string) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure()) //Dial op connection to the address
	if err != nil {
		log.Printf("Unable to connect to %s: %v", addr, err)
		return
	}

	node.mu.Lock()
	node.Peers[addr] = ras.NewRicartAgrawalaClient(conn) //Create a new HelloServiceclient and map it with its address.
	fmt.Println("Node(", node.Addr ,") has connected to Node(",addr, ")") //Print that the connection happened.
	node.mu.Unlock()
}

//Grpc endpoint.
func (node *Node) Request(ctx context.Context, req *ras.RequestMsg) (*ras.Ack, error) {
	fmt.Println("I am Node(", node.Id, ") and i just recieved a gRPC call from Node(", req.NodeId, ")")
	fmt.Println("...")
	node.HandleRequest(req)
	return &ras.Ack{Status: 200}, nil
}

func (node *Node) HandleRequest(req *ras.RequestMsg) {
	if node.state == HELD || 
				(node.state == WANTED && node.CompareRequest( req.LPTimestamp, req.NodeId)){
		node.ReqQueue = append(node.ReqQueue, req.Addr) //Tilføjer Node til vores reqQueue, da vi vil svare den senere
	} else {
		reqNode := Peers[req.Addr]
		reqNode.Response(&ras.ResponseMsg{LPTimestamp: node.LPTimestamp})
	}
}

func (node *Node) CompareRequest(timeStamp int32, id string) bool{
	if node.LPTimestamp < timeStamp {
		return true
	} else if node.LPTimestamp > timeStamp {
		return false
	} else if node.LPTimestamp == timeStamp {
		if node.Id < id {
			return true
		} else {
			return false
		}
	}
}