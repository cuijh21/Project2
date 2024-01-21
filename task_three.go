package main

import (
	// import necessary packages

	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"
)

func performTaskThree() {

	defer cleanupChildProcesses()

	// Start the wallet application
	go startProcess("wallet", "8080")

	// Start 5 nodes
	for i := 1; i <= 2; i++ {
		port := 3500 + i
		time.Sleep(1 * time.Second)
		go startProcess("node", fmt.Sprintf("%d", port))
	}

	go startProcess("consensus", "1111")

	// Ensure nodes are up and running
	time.Sleep(10 * time.Second)

	createCorruptedBlock()

	log.Println("Task Three Demo End.")

	cleanupChildProcesses()
}

func createCorruptedBlock() {

	// Assume this is the address of a known, trustworthy node
	nodeAddress := readKnownNodesFromFile("nodes.txt")[0]

	// Connect to the node
	client, err := rpc.Dial("tcp", nodeAddress)
	if err != nil {
		log.Fatalf("Failed to connect to the node: %v", err)
	}

	// Retrieve the current blockchain
	var currentBlockchain []*Block
	err = client.Call("Node.GetCurrentBlockchain", "request", &currentBlockchain)
	if err != nil {
		log.Fatalf("Failed to retrieve the blockchain: %v", err)
	}

	fmt.Println("Successfully retrieved the blockchain, current block count:", len(currentBlockchain))

	// Create a new corrupted block
	var lastBlock *Block
	if len(currentBlockchain) > 0 {
		lastBlock = currentBlockchain[len(currentBlockchain)-1]
	}
	newBlock := NewBlock([]*Transaction{}, lastBlock.Hash)
	CorruptBlock(newBlock) // Corrupt the block

	// Broadcast the corrupted block to all known nodes
	knownNodes := readKnownNodesFromFile("nodes.txt")
	for _, knownNode := range knownNodes {
		client, err := rpc.Dial("tcp", knownNode)
		if err != nil {
			log.Printf("Failed to connect to known node %s: %v", knownNode, err)
			continue
		}
		var reply string
		err = client.Call("Node.ReceiveNewBlock", newBlock, &reply)
		if err != nil {
			log.Printf("Failed to broadcast the corrupted block to node %s: %v", knownNode, err)
		} else {
			fmt.Printf("Node %s response: %s\n", knownNode, reply)
			client.Close()
			cleanupChildProcesses()
			os.Exit(0)
		}
		client.Close()
	}

	// Allow some time for the broadcast to be processed
	time.Sleep(5 * time.Second)
}

func CorruptBlock(block *Block) {
	// Change the previous hash of the block
	if len(block.PrevBlockHash) > 0 {
		block.PrevBlockHash[0] ^= 0xff
	}
}
