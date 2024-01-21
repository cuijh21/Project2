package main

import (
	"log"
	"os"
)

func startWalletApp(port string) {
	// First delete the genesis block file
	err := os.Remove(genesisBlockFile)
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to delete genesis block file: %v", err)
	}

	// First delete the nodes.txt file
	err = os.Remove("nodes.txt")
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to delete nodes.txt file: %v", err)
	}

	// Delete the consensus blockchain file
	err = os.Remove("consensus.blockchain")
	if err != nil && !os.IsNotExist(err) {
		log.Fatalf("Failed to delete consensus blockchain file: %v", err)
	}

	// Start the wallet application
	app := NewApplication()
	app.start(port)
}

func startBlockchainNode(port string) {
	blockchain := NewBlockchain() // Initialize the blockchain, loading or creating the genesis block

	nodeAddress := "127.0.0.1:" + port
	node := NewNode(nodeAddress, blockchain)

	// Write the node address to nodes.txt
	writeAddressToFile(nodeAddress, "nodes.txt")

	log.Printf("Node running at %s\n", nodeAddress)
	node.Start()
}

func writeAddressToFile(address, filename string) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	if _, err := file.WriteString(address + "\n"); err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: go run . [wallet|node|consensus|task] [num]")
	}

	mode := os.Args[1]
	num := os.Args[2]

	switch mode {
	case "wallet":
		startWalletApp(num)
	case "node":
		startBlockchainNode(num)
	case "consensus":
		consensus := NewConsensus()
		consensus.Start()
	case "task":
		if num == "one" {
			// Call the function for task one
			performTaskOne()
		} else if num == "three" {
			performTaskThree()
		} else if num == "four" {
			performTaskFour()
		} else if num == "five" {
			performTaskFive()
		} else {
			log.Fatal("Unknown task number: ", num)
		}

	default:
		log.Fatal("Unknown mode: ", mode)
	}
}
