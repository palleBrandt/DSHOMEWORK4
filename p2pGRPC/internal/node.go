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
	ras "p2p.com/proto"
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
	Id 					int32
	ResponseCounter 	int
	LPTimestamp 		int32
	mu 					sync.Mutex
	Peers 				map[string]ras.RicartAgrawalaServiceClient
	ReqQueue			[]string
	VZ					ras.VerbotenZoneServiceClient

	ras.UnimplementedRicartAgrawalaServiceServer //Denne her er nødvendig for at Node implementerer server interfacet genereret af protofilen.
}

//Denne metode "åbner" nodens server funtionalitet op. Definerer hvilket port noden er på.
//Den springer ud af skabet som server.
func (node *Node) StartListening() {
	lis, err := net.Listen("tcp", node.Addr) //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	ras.RegisterRicartAgrawalaServiceServer(grpcServer, node) //Dette registrerer noden som en værende en HelloServiceServer.

	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

//Dette er nodens "main" function.
func (node *Node) Start() error {
	node.ConnectToVZ()
	node.Peers = make(map[string]ras.RicartAgrawalaServiceClient) //Instantierer nodens map over peers.
	node.LPTimestamp = 0
	node.ResponseCounter = 0
	node.mu.Lock()
	node.state = RELEASED
	node.mu.Unlock()
	node.ReqQueue = node.ReqQueue[:0]
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
		time.Sleep(time.Duration(rand.Intn(3)) * time.Second)
		node.ResponseCounter = 0;
		node.LPTimestamp ++
		node.mu.Lock()
		node.state = WANTED;
		node.mu.Unlock()
		for key, Peer := range node.Peers {
			node.LPTimestamp ++;
			fmt.Println("Node(", node.Id, ") i am requesting at LPT(", node.LPTimestamp, ")")
			_, err := Peer.Request(context.Background(), &ras.RequestMsg{NodeId: node.Id, LPTimestamp: node.LPTimestamp, Addr: node.Addr})
			if err != nil {
				fmt.Println("Error making request to ", key)
			}
		}
		for {
			if node.ResponseCounter == len(node.Peers){
				node.LPTimestamp ++
				node.mu.Lock()
				node.state = HELD
				node.mu.Unlock()
				node.VZ.GoIn(context.Background(), &ras.VerbotenZoneMsg{Id: node.Id})
				time.Sleep(time.Duration(1) * time.Second)
				node.VZ.GoOut(context.Background(), &ras.VerbotenZoneMsg{Id: node.Id})
				node.LPTimestamp ++
				node.mu.Lock()
				node.state = RELEASED
				for _, addr := range node.ReqQueue{ //Runs through the possible requst there have been received while wanting the verboten-zone
					node.LPTimestamp ++;
					fmt.Println("Node(", node.Id, ") i am responding at LPT(", node.LPTimestamp, ")")
					node.Peers[addr].Response(context.Background(), &ras.ResponseMsg{LPTimestamp: node.LPTimestamp, NodeId: node.Id})
				}
				node.ReqQueue = make([]string, 0 , 10)
				node.mu.Unlock()
				// fmt.Println("Im not in the verboten-zone", node.Id, "at LPT", node.LPTimestamp)
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
	node.VZ = ras.NewVerbotenZoneServiceClient(conn)
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
	node.Peers[addr] = ras.NewRicartAgrawalaServiceClient(conn) //Create a new HelloServiceclient and map it with its address.
	fmt.Println("Node(", node.Addr ,") has connected to Node(",addr, ")") //Print that the connection happened.
	node.mu.Unlock()
}

//Grpc endpoint.
func (node *Node) Request(ctx context.Context, req *ras.RequestMsg) (*ras.Ack, error) {
	
	node.HandleRequest(req)
	return &ras.Ack{Status: 200}, nil
}

func (node *Node) HandleRequest(req *ras.RequestMsg) {
	node.mu.Lock()
	if node.state == HELD || 
				(node.state == WANTED && node.CompareRequest( req.LPTimestamp, req.NodeId)){
		node.ReqQueue = append(node.ReqQueue, req.Addr)
		node.LPTimestamp = MaxInt32(node.LPTimestamp, req.LPTimestamp) +1 //Tilføjer Node til vores reqQueue, da vi vil svare den senere
	} else {
		node.LPTimestamp = MaxInt32(node.LPTimestamp, req.LPTimestamp) +1
		reqNode := node.Peers[req.Addr]
		node.LPTimestamp ++
		reqNode.Response(context.Background(), &ras.ResponseMsg{LPTimestamp: node.LPTimestamp, NodeId: node.Id})
	}
	fmt.Println("I am Node(", node.Id, ") and i just recieved a gRPC request from Node(", req.NodeId, ") at LPT(", node.LPTimestamp, ")")
	fmt.Println("...")
	node.mu.Unlock()
}

func (node *Node) CompareRequest(timeStamp int32, id int32) bool{
	if node.LPTimestamp < timeStamp {
		return true
	} else if node.LPTimestamp > timeStamp {
		return false
	} else {
		if node.Id < id {
			return true
		} else {
			return false
		}
	} 
}

func (node *Node) Response(ctx context.Context, res *ras.ResponseMsg) (*ras.Ack, error){
	node.LPTimestamp = MaxInt32(node.LPTimestamp, res.LPTimestamp) +1
	fmt.Println("I am Node(", node.Id, ") and i just recieved a gRPC Response (", res.NodeId, ") at LPT(", node.LPTimestamp, ")")
	fmt.Println("...")
	node.mu.Lock()
	node.ResponseCounter ++
	node.mu.Unlock()
	return &ras.Ack{Status: 200}, nil
}

func MaxInt32(a,b int32) int32{
	if a > b {
		return a
	} else {
		return b
	}
}