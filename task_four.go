package main

import (
	// import necessary packages

	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"
)

func performTaskFour() {

	defer cleanupChildProcesses()

	// Start the wallet application
	go startProcess("wallet", "8080")

	// Start 2 nodes
	for i := 1; i <= 2; i++ {
		port := 5400 + i
		time.Sleep(1 * time.Second)
		go startProcess("node", fmt.Sprintf("%d", port))
	}

	go startProcess("consensus", "1111")

	// Ensure nodes are up and running
	time.Sleep(10 * time.Second)

	demonstrateInvalidPoWBlock()

	log.Println("Task Four Demo End.")

}

func demonstrateInvalidPoWBlock() {
	nodeAddress := readKnownNodesFromFile("nodes.txt")[0]

	client, err := rpc.Dial("tcp", nodeAddress)
	if err != nil {
		log.Fatalf("Failed to connect to the node: %v", err)
	}

	var currentBlockchain []*Block
	err = client.Call("Node.GetCurrentBlockchain", "request", &currentBlockchain)
	if err != nil {
		log.Fatalf("Failed to retrieve the blockchain: %v", err)
	}

	// 创建一个未正确解决 PoW 的区块
	invalidBlock := CreateInvalidPoWBlock(currentBlockchain[len(currentBlockchain)-1])

	// 广播这个无效的区块到所有已知节点
	knownNodes := readKnownNodesFromFile("nodes.txt")
	for _, knownNode := range knownNodes {
		client, err := rpc.Dial("tcp", knownNode)
		if err != nil {
			log.Printf("Failed to connect to known node %s: %v", knownNode, err)
			continue
		}
		var reply string
		err = client.Call("Node.ReceiveNewBlock", invalidBlock, &reply)
		if err != nil {
			log.Printf("Failed to broadcast the invalid PoW block to node %s: %v", knownNode, err)
		} else {
			fmt.Printf("Node %s response: %s\n", knownNode, reply)
			client.Close()
			cleanupChildProcesses()
			os.Exit(0)

		}
		client.Close()
	}

	time.Sleep(5 * time.Second)
}

func CreateInvalidPoWBlock(lastBlock *Block) *Block {
	invalidBlock := NewBlock([]*Transaction{}, lastBlock.Hash)

	// Intentionally create an invalid PoW for the block
	// you can change the block's hash without recalculating the nonce
	invalidBlock.Hash[0] ^= 0xff

	return invalidBlock
}
