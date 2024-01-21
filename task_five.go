package main

import (
	"fmt"
	"log"
	"net/rpc"
	"time"
)

func performTaskFive() {
	defer cleanupChildProcesses()

	// Start the wallet application
	go startProcess("wallet", "8080")

	// Start 2 nodes
	for i := 1; i <= 2; i++ {
		port := 9400 + i
		time.Sleep(1 * time.Second)
		go startProcess("node", fmt.Sprintf("%d", port))
	}
	// Ensure nodes are up and running
	time.Sleep(10 * time.Second)

	// 创建分叉
	createForkAndTest()

	log.Println("Task Five Demo End.")

}

func createForkAndTest() {
	// 读取已知节点的地址
	nodeAddresses := readKnownNodesFromFile("nodes.txt")
	if len(nodeAddresses) < 2 {
		log.Println("Not enough nodes to create a fork")
		return
	}

	// 假设前两个地址是我们需要操作的节点
	node1Address := nodeAddresses[0]
	node2Address := nodeAddresses[1]

	// 创建分叉
	createForkAtNode(node1Address)
	createForkAtNode(node2Address)

	// 等待一段时间，让网络中的节点同步和解决分叉
	time.Sleep(30 * time.Second)

	// 检查所有节点，确认它们是否都选择了最长的链
	checkNodesForLongestChain(nodeAddresses)
}

func createForkAtNode(nodeAddress string) {
	// 连接到节点
	client, err := rpc.Dial("tcp", nodeAddress)
	if err != nil {
		log.Printf("Failed to connect to node %s: %v", nodeAddress, err)
		return
	}
	defer client.Close()

	// 获取当前区块链的最新区块
	var blockchain []*Block
	err = client.Call("Node.GetCurrentBlockchain", "request", &blockchain)
	if err != nil {
		log.Printf("Failed to get the blockchain from node %s: %v", nodeAddress, err)
		return
	}

	// 获取最新区块
	var latestBlock *Block
	if len(blockchain) > 0 {
		latestBlock = blockchain[len(blockchain)-1]
	}

	// 创建并挖掘一个新区块
	newBlock := NewBlock([]*Transaction{}, latestBlock.Hash)

	// 广播新区块
	var reply string
	err = client.Call("Node.BroadcastNewBlock", newBlock, &reply)
	if err != nil {
		log.Printf("Failed to broadcast new block from node %s: %v", nodeAddress, err)
	}
}

func checkNodesForLongestChain(nodeAddresses []string) {
	longestChainLength := 0

	// 获取网络中最长的链长度
	for _, address := range nodeAddresses {
		client, err := rpc.Dial("tcp", address)
		if err != nil {
			log.Printf("Failed to connect to node %s: %v", address, err)
			continue
		}
		var blockchain []*Block
		err = client.Call("Node.GetCurrentBlockchain", "request", &blockchain)
		if err != nil {
			log.Printf("Failed to get blockchain from node %s: %v", address, err)
			client.Close()
			continue
		}
		client.Close()
		if len(blockchain) > longestChainLength {
			longestChainLength = len(blockchain)
		}
	}

	// 检查每个节点是否遵循了最长链规则
	for _, address := range nodeAddresses {
		client, err := rpc.Dial("tcp", address)
		if err != nil {
			log.Printf("Failed to connect to node %s: %v", address, err)
			continue
		}
		var blockchain []*Block
		err = client.Call("Node.GetCurrentBlockchain", "request", &blockchain)
		if err != nil {
			log.Printf("Failed to get blockchain from node %s: %v", address, err)
			client.Close()
			continue
		}
		client.Close()
		if len(blockchain) != longestChainLength {
			log.Printf("Node %s is not on the longest chain", address)
		} else {
			log.Printf("Node %s is on the longest chain", address)
		}
	}
}
