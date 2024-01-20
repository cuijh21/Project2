package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type Blockchain struct {
	Blocks  []*Block
	Mempool *Mempool
}

// NewBlockchain creates a new blockchain with the initial genesis block
func NewBlockchain() *Blockchain {
	genesisBlock := LoadGenesisBlock()
	if genesisBlock == nil {
		genesisBlock = NewGenesisBlock()
		SaveGenesisBlock(genesisBlock)
	}

	return &Blockchain{
		Blocks:  []*Block{genesisBlock},
		Mempool: NewMempool(),
	}
}

// NewGenesisBlock creates a genesis block
func NewGenesisBlock() *Block {
	genesisTransactions := make([]*Transaction, 0)

	// First delete the existing users.txt file
	err := os.Remove("users.txt")
	if err != nil && !os.IsNotExist(err) {
		log.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		username, password := generateRandomCredentials()
		wallet := NewWallet()
		saveGenesisUserToFile(username, password, wallet.Address())

		genesisTransactions = append(genesisTransactions, NewTransaction("", wallet.Address(), 100))
	}

	return NewBlock(genesisTransactions, []byte{})
}

// NewGenesisTransaction creates the initial transaction for the genesis block
func NewGenesisTransaction() *Transaction {
	// The genesis block's transaction can have special markings, like empty From and To
	tx := NewTransaction("", "", 0) // Genesis block transactions might not have actual "from", "to" or "amount"
	return tx
}

// IsValid checks if the block's hash is correct
func (b *Block) IsValid() bool {
	if !bytes.Equal(b.Hash, b.ComputeHash()) {
		fmt.Println("invalid hash")
	}
	return bytes.Equal(b.Hash, b.ComputeHash())
}

// ValidateChain validates the blockchain
func (bc *Blockchain) ValidateChain() bool {
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if !bytes.Equal(currentBlock.Hash, currentBlock.ComputeHash()) {
			fmt.Printf("Block %d has invalid hash\n", i)
			return false
		}

		if !bytes.Equal(currentBlock.PrevBlockHash, prevBlock.Hash) {
			fmt.Printf("Block %d points to incorrect previous hash\n", i)
			return false
		}
	}
	return true
}

// AddTransactionToMempool adds a transaction to the mempool
func (bc *Blockchain) AddTransactionToMempool(tx *Transaction) {
	if tx.IsValid() && bc.IsValidAddress(tx.From) && bc.IsValidAddress(tx.To) && bc.GetBalance(tx.From) >= tx.Amount {
		bc.Mempool.AddTransaction(tx)
	} else {
		fmt.Println("Invalid transaction, invalid addresses or insufficient balance")
	}
}

// MineBlock mines a block from transactions in the mempool
func (bc *Blockchain) MineBlock() {
	lastBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := NewBlock(bc.Mempool.GetTransactions(), lastBlock.Hash)
	bc.Blocks = append(bc.Blocks, newBlock)
	bc.Mempool.Clear()
}

// GetBalance calculates and returns the balance for a given address
func (bc *Blockchain) GetBalance(address string) int {
	balance := 0
	for _, block := range bc.Blocks {
		for _, tx := range block.Transactions {
			if tx.From == address {
				balance -= tx.Amount
			}
			if tx.To == address {
				balance += tx.Amount
			}
		}
	}
	return balance
}

// IsValidAddress checks if an address is in a valid format
func (bc *Blockchain) IsValidAddress(address string) bool {
	// Example: check the length of the address
	if !(len(address) > 0) {
		fmt.Println("Error: length of address should be greater than 0 ")
	}
	return len(address) > 0 // More complex logic can be added here, such as checking if the address conforms to a specific format
}

// PrintBlockchain displays detailed information of all blocks
func (bc *Blockchain) PrintBlockchain() {
	for _, block := range bc.Blocks {
		jsonBlock, err := json.MarshalIndent(block, "", "  ")
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			continue
		}
		fmt.Println(string(jsonBlock))
	}
}

// saveGenesisUserToFile saves user information to a file
func saveGenesisUserToFile(username, password, address string) {
	file, err := os.OpenFile("users.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("%s:%s:%s\n", username, password, address))
	if err != nil {
		log.Fatal(err)
	}
}

const genesisBlockFile = "genesis.block"

// SaveGenesisBlock saves the genesis block to a file
func SaveGenesisBlock(block *Block) {
	file, err := os.Create(genesisBlockFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(block); err != nil {
		log.Fatal(err)
	}
}

// LoadGenesisBlock loads the genesis block from a file
func LoadGenesisBlock() *Block {
	file, err := os.Open(genesisBlockFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File does not exist, return nil
		}
		log.Fatal(err)
	}
	defer file.Close()

	var block Block
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&block); err != nil {
		log.Fatal(err)
	}
	return &block
}
