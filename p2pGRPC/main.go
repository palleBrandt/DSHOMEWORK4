package main

import "p2p.com/internal"


func main() {
	node1 := node.Node{Addr: "localhost:50051", Id: 1}
	node2 := node.Node{Addr: "localhost:50052", Id: 2}
	node3 := node.Node{Addr: "localhost:50053", Id: 3}

	go runNode(node1)
	go runNode(node2)
	go runNode(node3)

	for {}
	
}

func runNode(node node.Node) {
	node.Start()
}