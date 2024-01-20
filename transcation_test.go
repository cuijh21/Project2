package main

import (
	"reflect"
	"testing"
)

func TestNewTransaction(t *testing.T) {
	tx := NewTransaction("from", "to", 10)

	if tx.From != "from" || tx.To != "to" || tx.Amount != 10 {
		t.Error("NewTransaction() failed to create a transaction with correct data")
	}
}

func TestTransactionHash(t *testing.T) {
	tx := NewTransaction("from", "to", 10)
	hash := tx.Hash()

	if len(hash) == 0 {
		t.Error("Hash() failed to generate a hash for the transaction")
	}
}

func TestTransactionSerializeDeserialize(t *testing.T) {
	tx := NewTransaction("from", "to", 10)
	serialized, err := tx.Serialize()
	if err != nil {
		t.Errorf("Serialize() failed with error: %v", err)
	}

	deserialized, err := DeserializeTransaction(serialized)
	if err != nil {
		t.Errorf("DeserializeTransaction() failed with error: %v", err)
	}

	if !reflect.DeepEqual(tx, deserialized) {
		t.Error("Serialize() and DeserializeTransaction() failed, the transactions are not equal")
	}
}
