package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const (
	consensusFile   = "consensus.blockchain" // File name for storing the blockchain consensus data
	nodeAddressFile = "nodes.txt"            // File name for storing known node addresses
	pollInterval    = 3 * time.Second        // Interval for polling updates in the blockchain network
)

type Consensus struct {
	mutex             sync.Mutex
	Blockchain        *Blockchain
	KnownNodes        []string
	KnownTransactions map[string]bool // Stores hashes of known transactions
}

// NewConsensus initializes a new consensus mechanism
func NewConsensus() *Consensus {
	c := &Consensus{
		Blockchain:        NewBlockchain(),
		KnownNodes:        readKnownNodesFromFile(nodeAddressFile),
		KnownTransactions: make(map[string]bool),
	}

	// Load initial blockchain data
	c.loadInitialBlockchain()

	return c
}

// loadInitialBlockchain loads the genesis block from a file
func (c *Consensus) loadInitialBlockchain() {
	file, err := os.ReadFile(genesisBlockFile)
	if err != nil {
		log.Printf("Error reading genesis block file: %v", err)
		return
	}

	var genesisBlock Block
	if err := json.Unmarshal(file, &genesisBlock); err != nil {
		log.Printf("Error unmarshalling genesis block: %v", err)
		return
	}

	c.Blockchain.Blocks = append(c.Blockchain.Blocks, &genesisBlock)
}

// readKnownNodesFromFile reads node addresses from a file
func readKnownNodesFromFile(filename string) []string {
	var nodes []string
	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening node address file: %v", err)
		return nodes
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nodes = append(nodes, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from node address file: %v", err)
	}

	return nodes
}

// Start begins the consensus process
func (c *Consensus) Start() {
	ticker := time.NewTicker(pollInterval)
	for {
		select {
		case <-ticker.C:
			c.UpdateBlockchain()
		}
	}
}

// UpdateBlockchain updates the blockchain based on network consensus
func (c *Consensus) UpdateBlockchain() {
	var longestChain []*Block
	var longestChainLength int
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for _, node := range c.KnownNodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", node)
			if err != nil {
				log.Printf("Unable to connect to node %s: %v", node, err)
				return
			}
			defer client.Close()

			var reply []*Block
			err = client.Call("Node.GetCurrentBlockchain", "consensus", &reply)
			if err != nil {
				log.Printf("Error requesting blockchain from node %s: %v", node, err)
				return
			}

			// Pass KnownTransactions when calling isValidChain
			mutex.Lock()
			if isValidChain(reply, c.KnownTransactions) && len(reply) > longestChainLength {
				longestChain = reply
				longestChainLength = len(reply)
			}
			mutex.Unlock()
		}(node)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Check whether to update the blockchain after all goroutines have completed
	if longestChain != nil {
		c.SaveBlockchain(longestChain)
	}
}

// isValidChain checks if a blockchain is valid
func isValidChain(chain []*Block, knownTransactions map[string]bool) bool {
	// Implement logic to check the validity of the blockchain here
	// For example, check the hash of blocks, validity of transactions, etc.
	// Below is a simplified example
	for i := 0; i < len(chain)-1; i++ {
		if !bytes.Equal(chain[i].Hash, chain[i+1].PrevBlockHash) {
			return false
		}
	}

	// Check for duplicate transactions
	for _, block := range chain {
		for _, tx := range block.Transactions {
			txID := string(tx.ID)
			if knownTransactions[txID] {
				// Return false if a duplicate transaction is found
				return false
			}
			knownTransactions[txID] = true
		}
	}

	return true
}

// SaveBlockchain saves the blockchain to a file
func (c *Consensus) SaveBlockchain(blocks []*Block) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Reset the known transactions set
	c.KnownTransactions = make(map[string]bool)

	file, err := os.Create(consensusFile)
	if err != nil {
		log.Printf("Unable to create consensus file: %v", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(blocks); err != nil {
		log.Printf("Error encoding blockchain: %v", err)
	}
}
