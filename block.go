package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

const targetBits = 3              // Mining difficulty definition
const maxNonce = 1<<31 - 1        // Maximum value for a 32-bit integer, used as the limit for mining attempts
const MaxTransactionsPerBlock = 5 // Assuming a maximum of 5 transactions per block

type Block struct {
	Timestamp     int64
	Transactions  []*Transaction // Replacing the original Data field with a list of transactions
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

// SetHash calculates and sets the hash of the block, without returning a value
func (b *Block) SetHash() {
	data := prepareData(b, b.Nonce)
	hash := sha256.Sum256(data)
	b.Hash = hash[:]
}

// ComputeHash calculates and returns the hash of the block
func (b *Block) ComputeHash() []byte {
	data := prepareData(b, b.Nonce)
	hash := sha256.Sum256(data)
	return hash[:]
}

// hashTransactions serializes a slice of transactions and calculates its hash
func hashTransactions(transactions []*Transaction) []byte {
	var txHashes [][]byte

	for _, tx := range transactions {
		serializedTx, err := tx.Serialize()
		if err != nil {
			panic(err) // In real applications, errors should be handled more gracefully
		}
		txHashes = append(txHashes, serializedTx)
	}

	txHash := sha256.Sum256(bytes.Join(txHashes, []byte{}))
	return txHash[:] // Converts the array to a slice
}

func IntToHex(n int64) []byte {
	return []byte(strconv.FormatInt(n, 16))
}

// MineBlock implements the mining functionality
func (b *Block) MineBlock() {
	var hashInt big.Int
	var hash [32]byte
	nonce := 0

	fmt.Println("Mining a new block...")

	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))

	for nonce < maxNonce {
		data := prepareData(b, nonce)
		hash = sha256.Sum256(data)
		fmt.Printf("\r%x", hash)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			fmt.Println("\n\nSuccess!")
			b.Hash = hash[:]
			b.Nonce = nonce
			break
		} else {
			nonce++
		}
	}
}

func prepareData(b *Block, nonce int) []byte {
	return bytes.Join(
		[][]byte{
			b.PrevBlockHash,
			hashTransactions(b.Transactions), // Hash of the transaction data
			IntToHex(b.Timestamp),
			IntToHex(int64(targetBits)),
			IntToHex(int64(nonce)),
		},
		[]byte{},
	)
}

// Note, the NewBlock function no longer needs to create a genesis block, it's only for creating regular blocks
func NewBlock(transactions []*Transaction, prevBlockHash []byte) *Block {
	// Ensure the number of transactions in the block doesn't exceed the limit
	if len(transactions) > MaxTransactionsPerBlock {
		transactions = transactions[:MaxTransactionsPerBlock]
	}

	block := &Block{Timestamp: time.Now().Unix(), Transactions: transactions, PrevBlockHash: prevBlockHash, Hash: []byte{}, Nonce: 0}
	block.MineBlock() // Mine all non-genesis blocks
	return block
}
