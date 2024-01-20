package main

import (
	"testing"
)

func TestNewBlockchain(t *testing.T) {
	blockchain := NewBlockchain()

	if len(blockchain.Blocks) != 1 {
		t.Errorf("NewBlockchain() failed, expected blockchain length of 1, got %v", len(blockchain.Blocks))
	}
}

func TestValidateChain(t *testing.T) {
	blockchain := NewBlockchain()

	// 创建并添加一个包含交易的区块
	transaction := NewTransaction("from", "to", 50)
	newBlock := NewBlock([]*Transaction{transaction}, blockchain.Blocks[len(blockchain.Blocks)-1].Hash)
	blockchain.Blocks = append(blockchain.Blocks, newBlock)
	// blockchain.PrintBlockchain()
	// 验证区块链是否有效
	if !blockchain.ValidateChain() {
		t.Error("ValidateChain() failed, the chain should be valid")
	}

	// 篡改区块链使其无效
	blockchain.Blocks[1].Transactions = []*Transaction{NewTransaction("from", "to", 100)}
	if blockchain.ValidateChain() {
		t.Error("ValidateChain() failed, the chain should be invalid after tampering")
	}
}

// 等回头,用户有可用余额的时候, 在验证这个test用例
// func TestAddTransactionToMempool(t *testing.T) {
// 	blockchain := NewBlockchain()
// 	tx := NewTransaction("from", "to", 10)
// 	blockchain.AddTransactionToMempool(tx)

// 	if len(blockchain.Mempool.Transactions) != 1 {
// 		t.Error("AddTransactionToMempool() failed, the transaction was not added to the mempool")
// 	}
// }
