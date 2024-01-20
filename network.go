package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
)

// Node represents a node in the blockchain network
type Node struct {
	Address               string
	Blockchain            *Blockchain
	KnownNodes            []string
	BlocksInSync          map[string]bool
	mutex                 sync.Mutex
	BlockchainPool        chan [][]*Block // 用于存储多个区块链版本
	PollInterval          int             // 同步区块链的时间间隔（分钟）
	ProcessedTransactions map[string]bool // 新增字段，用于存储已处理的交易ID
}

// NewNode creates a new Node instance with a specified poll interval
func NewNode(address string, blockchain *Blockchain, pollInterval int) *Node {
	return &Node{
		Address:               address,
		Blockchain:            blockchain,
		KnownNodes:            []string{},
		BlocksInSync:          make(map[string]bool),
		BlockchainPool:        make(chan [][]*Block, 100), // 假设池大小为100
		PollInterval:          pollInterval,               // 设置同步区块链的时间间隔
		ProcessedTransactions: make(map[string]bool),      // 初始化
	}
}

// Start initializes the node's server and connects to known nodes
func (node *Node) Start() {
	node.registerRPCMethods()
	listener, err := net.Listen("tcp", node.Address)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	for _, knownNode := range node.KnownNodes {
		node.connectToNode(knownNode)
	}

	fmt.Printf("Node %s is listening on %s\n", node.Address, listener.Addr())
	// fmt.Println("initial blockchain is:")
	// node.Blockchain.PrintBlockchain()

	// 启动一个协程开始挖矿
	go node.StartMining()
	// 启动一个协程更新自身区块
	go node.StartUpdatingChain()

	// 启动一个协程开始同步区块链
	go node.StartBlockchainSync()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

// connectToNode handles the connection to another node
func (node *Node) connectToNode(address string) {
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		log.Printf("Unable to connect to node %s: %v", address, err)
		return
	}
	defer client.Close()

	// Example RPC call
	var reply string
	err = client.Call("Node.Ping", node.Address, &reply)
	if err != nil {
		log.Printf("Error calling Node.Ping: %v", err)
		return
	}

	fmt.Printf("Ping to %s successful: %s\n", address, reply)
}

// registerRPCMethods registers methods that can be called remotely
func (node *Node) registerRPCMethods() {
	rpc.Register(node)
}

// Ping is an example RPC method
func (node *Node) Ping(address string, reply *string) error {
	*reply = fmt.Sprintf("Pong to %s from %s", address, node.Address)
	return nil
}

// AddKnownNode adds a new node to the list of known nodes
func (node *Node) AddKnownNode(address string) {
	node.mutex.Lock()
	defer node.mutex.Unlock()

	for _, knownNode := range node.KnownNodes {
		if knownNode == address {
			return
		}
	}
	node.KnownNodes = append(node.KnownNodes, address)
}

// ReceiveTransaction receives a transaction broadcasted from other nodes.
func (node *Node) ReceiveTransaction(tx *Transaction, reply *string) error {
	txID := string(tx.ID) // 或者使用更适合的方式来转换/格式化交易ID

	// 检查交易是否已被处理
	node.mutex.Lock()
	if node.ProcessedTransactions[txID] {
		node.mutex.Unlock()
		*reply = "Transaction already processed"
		return nil
	}

	// 标记交易为已处理
	node.ProcessedTransactions[txID] = true
	node.mutex.Unlock()

	// 现有的处理逻辑
	if !node.TransactionExists(tx) {
		node.Blockchain.AddTransactionToMempool(tx)
		*reply = "Transaction received and added to the mempool"
	} else {
		*reply = "Transaction already exists"
	}
	return nil
}

// TransactionExists checks if a transaction already exists in the blockchain or the mempool.
func (node *Node) TransactionExists(tx *Transaction) bool {
	// 检查交易池
	for _, mempoolTx := range node.Blockchain.Mempool.Transactions {
		if bytes.Equal(mempoolTx.ID, tx.ID) {
			return true
		}
	}

	// 检查已经确认的区块
	for _, block := range node.Blockchain.Blocks {
		for _, blockTx := range block.Transactions {
			if bytes.Equal(blockTx.ID, tx.ID) {
				return true
			}
		}
	}

	return false
}

// StartMining starts the mining process.
func (node *Node) StartMining() {
	for {
		// check has transcation in mempool
		if len(node.Blockchain.Mempool.Transactions) > 0 {
			fmt.Println("new transcation, start mining...")

			// get transcation from mempool
			newBlock := NewBlock(node.Blockchain.Mempool.Transactions, node.Blockchain.GetLatestBlock().Hash)
			newBlock.MineBlock() // mine block

			// 将挖掘出的新区块添加到区块链
			node.Blockchain.Blocks = append(node.Blockchain.Blocks, newBlock)
			fmt.Println("end mining...")

			// 清空交易池
			node.Blockchain.Mempool.Clear()
			fmt.Println("clear transcation mempool...")

			// 挖矿成功后广播整个区块链
			node.BroadcastNewBlock() // 使用新的广播函数
			fmt.Println("broadcast newblock...")

		}
		// simple delay to limit the mining speed
		time.Sleep(10 * time.Second)
	}
}

// BroadcastNewBlock broadcasts the current node's entire blockchain.
func (node *Node) BroadcastNewBlock() {
	// 从 nodes.txt 文件中读取已知节点列表
	knownNodes, err := node.ReadKnownNodesFromFile("nodes.txt")
	if err != nil {
		log.Printf("Error reading known nodes from file: %v", err)
		return
	}

	for _, knownNode := range knownNodes {
		go func(knownNode string) {
			client, err := rpc.Dial("tcp", knownNode)
			if err != nil {
				log.Printf("Error dialing known node %s: %v", knownNode, err)
				return
			}
			defer client.Close()

			var reply string
			err = client.Call("Node.ReceiveNewBlock", node.Blockchain.Blocks, &reply)
			if err != nil {
				log.Printf("Error broadcasting blockchain to node %s: %v", knownNode, err)
			}
		}(knownNode)
	}
}

// ReadKnownNodesFromFile reads a list of known nodes from a file.
func (node *Node) ReadKnownNodesFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var nodes []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		nodes = append(nodes, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nodes, nil
}

// StartUpdatingChain updates the blockchain.
func (node *Node) StartUpdatingChain() {
	for blockchainVersions := range node.BlockchainPool {
		longestChain := node.Blockchain.Blocks
		for _, chain := range blockchainVersions {
			if len(chain) > len(longestChain) && node.IsValidChain(chain) {
				longestChain = chain
			}
		}
		node.mutex.Lock()
		node.Blockchain.Blocks = longestChain
		node.mutex.Unlock()

		node.Blockchain.PrintBlockchain()
		fmt.Println("Blockchain updated with longer valid chain")
	}
}

// IsValidChain validates the entire blockchain.
func (node *Node) IsValidChain(chain []*Block) bool {
	// check validation of chain
	for i, block := range chain {
		if i == 0 {
			continue // skip the geneisblock
		}
		if !block.IsValid() || !bytes.Equal(block.PrevBlockHash, chain[i-1].Hash) {
			return false
		}
	}
	return true
}

// AreChainsEqual compares two blockchains to see if they are the same.
func (node *Node) AreChainsEqual(chain1, chain2 []*Block) bool {
	if len(chain1) != len(chain2) {
		return false
	}

	for i := range chain1 {
		if !bytes.Equal(chain1[i].Hash, chain2[i].Hash) {
			return false
		}
	}

	return true
}

// ReceiveNewBlock receives a version of the blockchain and appends it to the BlockchainPool.
func (node *Node) ReceiveNewBlock(newBlocks []*Block, reply *string) error {
	fmt.Println("recieved new blockchain copy...")
	node.BlockchainPool <- [][]*Block{newBlocks}
	*reply = "New blockchain version added to pool"
	return nil
}

// FindForkIndex finds the fork point between the current chain and a new chain.
func (node *Node) FindForkIndex(newBlocks []*Block) int {
	for i, block := range node.Blockchain.Blocks {
		if !bytes.Equal(block.Hash, newBlocks[i].Hash) {
			return i
		}
	}
	return -1
}

// IsValidChainFromIndex validates the chain from a specified index.
func (node *Node) IsValidChainFromIndex(blocks []*Block, index int) bool {
	// if index invalid, then return false
	if index < 0 || index >= len(blocks) {
		return false
	}

	for _, block := range blocks[index:] {
		if !block.IsValid() {
			return false
		}
	}
	return true
}

// GetLatestBlock gets the last block in the blockchain.
func (bc *Blockchain) GetLatestBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

// GetCurrentBlockchain provides a copy of the node's current blockchain.
func (node *Node) GetCurrentBlockchain(request string, reply *[]*Block) error {
	fmt.Println("有节点正在获取最新的副本..")
	*reply = node.Blockchain.Blocks
	return nil
}

// RequestLatestBlockchain requests the latest blockchain data from other nodes.
func (node *Node) RequestLatestBlockchain() {

	knownNodes, err := node.ReadKnownNodesFromFile("nodes.txt")
	if err != nil {
		log.Printf("Error reading known nodes from file: %v", err)
	} else {
		node.KnownNodes = knownNodes
	}

	for _, knownNode := range node.KnownNodes {
		log.Printf("node%s", knownNode)
		go func(knownNode string) {
			client, err := rpc.Dial("tcp", knownNode)
			if err != nil {
				log.Printf("Unable to connect to known node %s: %v", knownNode, err)
				return
			}
			defer client.Close()

			var reply []*Block
			err = client.Call("Node.GetCurrentBlockchain", node.Address, &reply)
			if err != nil {
				log.Printf("Error requesting blockchain from node %s: %v", knownNode, err)
				return
			}

			// send chain to BlockchainPool
			node.BlockchainPool <- [][]*Block{reply}
		}(knownNode)
	}
}

// StartBlockchainSync starts a task to periodically sync the blockchain.
func (node *Node) StartBlockchainSync() {
	ticker := time.NewTicker(time.Duration(node.PollInterval) * time.Second) // 使用秒作为单位
	for {
		select {
		case <-ticker.C:
			node.RequestLatestBlockchain()
		}
	}
}
