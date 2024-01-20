package main

import (
	"crypto/sha256"
	"encoding/json"
	"time"
)

type Transaction struct {
	ID        []byte    // Transaction ID
	From      string    // Sender's address
	To        string    // Receiver's address
	Amount    int       // Transaction amount
	Timestamp time.Time // Transaction creation time
}

// NewTransaction creates a new transaction.
func NewTransaction(from, to string, amount int) *Transaction {
	tx := Transaction{
		ID:        nil,
		From:      from,
		To:        to,
		Amount:    amount,
		Timestamp: time.Now(), // Set the current time as the transaction creation time
	}
	tx.ID = tx.Hash()
	return &tx
}

// Hash generates the hash of the transaction.
func (tx *Transaction) Hash() []byte {
	var hash [32]byte
	txCopy := *tx
	txCopy.ID = []byte{}

	encoded, err := json.Marshal(txCopy)
	if err != nil {
		panic(err) // In real applications, you might want to handle errors more gracefully
	}

	hash = sha256.Sum256(encoded)
	return hash[:]
}

// Serialize serializes the Transaction using JSON.
func (tx *Transaction) Serialize() ([]byte, error) {
	return json.Marshal(tx)
}

// DeserializeTransaction deserializes a byte sequence using JSON to restore a Transaction object.
func DeserializeTransaction(data []byte) (*Transaction, error) {
	var transaction Transaction
	err := json.Unmarshal(data, &transaction)
	if err != nil {
		return nil, err
	}
	return &transaction, nil
}

// IsValid performs a simple validation: ensuring the amount is not negative and both sender and receiver are not empty.
func (tx *Transaction) IsValid() bool {
	return tx.Amount >= 0 && tx.From != "" && tx.To != ""
}
