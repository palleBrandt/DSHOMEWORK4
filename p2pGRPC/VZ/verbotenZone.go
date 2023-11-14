package verbotenZone

import (

	"net"
	"sync"
	"fmt"
	"log"

	"golang.org/x/net/context"
	TRS "p2p.com/proto"
	"google.golang.org/grpc"
)

type VerbotenZone struct {
	//Maakes it possible to lock the server
	sync.Mutex
    // an interface that the server type needs to have
    TRS.UnimplementedVerbotenZoneServiceServer

	// A list of all streams created between the clients and the server
	t int32;
}

func (vz *VerbotenZone) StartListening() {
	lis, err := net.Listen("tcp", "localhost:50054") //Listener på denne addr
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	TRS.RegisterVerbotenZoneServiceServer(grpcServer, vz) //Dette registrerer noden som en værende en HelloServiceServer.

	// Start listening for incoming connections
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

//Dette er nodens "main" function.
func (vz *VerbotenZone) Start() error {
	vz.t = 0;
	go vz.StartListening() //Go routine med kald til "server" funktionaliteten.
	for{}
}

func (vz *VerbotenZone) GoIn(ctx context.Context, vzMsg *TRS.VerbotenZoneMsg) (*TRS.Ack, error){
	vz.t ++
	fmt.Println("VZ: Node ", vzMsg.Id, "accessed the VerbotenZone at time", vz.t)
	return &TRS.Ack{Status: 200}, nil
}

func (vz *VerbotenZone) GoOut(ctx context.Context, vzMsg *TRS.VerbotenZoneMsg) (*TRS.Ack, error){
	vz.t ++
	fmt.Println("VZ: Node ", vzMsg.Id, "left the VerbotenZone at time", vz.t)
	return &TRS.Ack{Status: 200}, nil
}