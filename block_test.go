package main

import (
	"math/big"
	"testing"
)

func TestSetHash(t *testing.T) {
	block := Block{Timestamp: 0, Transactions: []*Transaction{}, PrevBlockHash: []byte{}, Hash: []byte{}, Nonce: 0}
	block.SetHash()

	if len(block.Hash) == 0 {
		t.Error("SetHash() failed to set the Hash of the block")
	}
}

func TestComputeHash(t *testing.T) {
	block := Block{Timestamp: 0, Transactions: []*Transaction{}, PrevBlockHash: []byte{}, Hash: []byte{}, Nonce: 0}
	hash := block.ComputeHash()

	if len(hash) == 0 {
		t.Error("ComputeHash() failed to compute the Hash of the block")
	}
}

func TestMineBlock(t *testing.T) {
	block := Block{Timestamp: 0, Transactions: []*Transaction{}, PrevBlockHash: []byte{}, Hash: []byte{}, Nonce: 0}
	block.MineBlock()

	var hashInt big.Int
	hashInt.SetBytes(block.Hash)

	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	if hashInt.Cmp(target) >= 0 {
		t.Errorf("MineBlock() failed to mine a block with hash less than the target")
	}
}
