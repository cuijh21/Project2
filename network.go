package main

import (
	"bytes"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

// Node represents a node in the blockchain network
type Node struct {
	Address         string
	Blockchain      *Blockchain
	BlockchainMutex sync.Mutex
}

// NewNode creates a new Node instance
func NewNode(address string, blockchain *Blockchain) *Node {
	return &Node{
		Address:    address,
		Blockchain: blockchain,
	}
}

func (node *Node) Start() {
	rpc.Register(node)
	listener, err := net.Listen("tcp", node.Address)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// 启动一个协程来开始挖掘
	go node.StartMining()

	// 启动定时同步任务
	go func() {
		syncTicker := time.NewTicker(30 * time.Second) // 每30秒同步一次
		for {
			select {
			case <-syncTicker.C:
				node.SyncWithNetwork()
			}
		}
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

func (node *Node) AddBlockToBlockchain(block *Block) {
	node.BlockchainMutex.Lock()
	defer node.BlockchainMutex.Unlock()

	if block.PrevBlockHash == nil || bytes.Equal(node.Blockchain.GetLatestBlock().Hash, block.PrevBlockHash) {
		node.Blockchain.Blocks = append(node.Blockchain.Blocks, block)
		node.Blockchain.Mempool.Clear() // Clear mempool after adding a block
	}
}

func (node *Node) ReceiveNewBlock(block *Block, reply *string) error {
	if block.IsValid() {
		node.AddBlockToBlockchain(block)
		*reply = "Block added to the blockchain"
	} else {
		*reply = "Invalid block"
		log.Println("Received invalid block, rejecting")
	}
	return nil
}

func (node *Node) BroadcastNewBlock(block *Block, reply *string) error {
	KnownNodes := readKnownNodesFromFile("nodes.txt")

	for _, knownNode := range KnownNodes {
		go func(knownNode string) {
			client, err := rpc.Dial("tcp", knownNode)
			if err != nil {
				log.Printf("Error dialing known node %s: %v", knownNode, err)
				return
			}
			defer client.Close()

			var nodeReply string
			err = client.Call("Node.ReceiveNewBlock", block, &nodeReply)
			if err != nil {
				log.Printf("Error broadcasting new block to node %s: %v", knownNode, err)
			}
		}(knownNode)
	}
	*reply = "Broadcast initiated"
	return nil
}

func (bc *Blockchain) GetLatestBlock() *Block {
	if len(bc.Blocks) == 0 {
		return nil
	}
	return bc.Blocks[len(bc.Blocks)-1]
}

func (node *Node) ReceiveTransaction(tx *Transaction, reply *string) error {
	if tx.IsValid() {
		// 将交易添加到交易池
		node.Blockchain.AddTransactionToMempool(tx)
		*reply = "Transaction added to mempool"
	} else {
		*reply = "Invalid transaction"
	}
	return nil
}

func (node *Node) MineBlockFromMempool() {
	node.BlockchainMutex.Lock()
	defer node.BlockchainMutex.Unlock()

	// 检查交易池是否有待处理的交易
	if len(node.Blockchain.Mempool.Transactions) > 0 {
		// 取出一个交易来挖掘新区块
		transaction := node.Blockchain.Mempool.Transactions[0]
		newBlock := NewBlock([]*Transaction{transaction}, node.Blockchain.GetLatestBlock().Hash)

		// 挖掘新区块
		newBlock.MineBlock()

		// 将新区块添加到区块链
		node.Blockchain.Blocks = append(node.Blockchain.Blocks, newBlock)

		// 从交易池中移除已处理的交易
		node.Blockchain.Mempool.Transactions = node.Blockchain.Mempool.Transactions[1:]

		// 广播新区块
		var reply string
		err := node.BroadcastNewBlock(newBlock, &reply)
		if err != nil {
			log.Printf("Failed to broadcast new block from node %v", err)
		}
	}
}

func (node *Node) StartMining() {
	for {
		node.MineBlockFromMempool()
		time.Sleep(2 * time.Second) // 为简化起见，我们在这里设置了一个固定的延迟
	}
}

func (node *Node) GetCurrentBlockchain(request string, reply *[]*Block) error {
	node.BlockchainMutex.Lock()
	defer node.BlockchainMutex.Unlock()

	*reply = node.Blockchain.Blocks
	return nil
}

func (node *Node) UpdateLocalBlockchain(newBlocks []*Block) {
	node.BlockchainMutex.Lock()
	defer node.BlockchainMutex.Unlock()

	if len(newBlocks) > len(node.Blockchain.Blocks) {
		node.Blockchain.Blocks = newBlocks
	}
}

func (node *Node) SyncWithNetwork() {
	knownNodes := readKnownNodesFromFile("nodes.txt")

	for _, knownNode := range knownNodes {
		go func(knownNode string) {
			client, err := rpc.Dial("tcp", knownNode)
			if err != nil {
				log.Printf("Error dialing known node %s: %v", knownNode, err)
				return
			}
			defer client.Close()

			var remoteBlocks []*Block
			err = client.Call("Node.GetCurrentBlockchain", "sync_request", &remoteBlocks)
			if err != nil {
				log.Printf("Error getting blockchain from node %s: %v", knownNode, err)
				return
			}

			node.UpdateLocalBlockchain(remoteBlocks)
		}(knownNode)
	}
}
