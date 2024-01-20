package main

import (
	"reflect"
	"testing"
)

func TestMempoolOperations(t *testing.T) {
	mempool := NewMempool()
	tx1 := NewTransaction("from1", "to1", 10)
	tx2 := NewTransaction("from2", "to2", 20)

	mempool.AddTransaction(tx1)
	mempool.AddTransaction(tx2)

	if len(mempool.Transactions) != 2 {
		t.Errorf("AddTransaction() failed, expected 2 transactions in mempool, got %v", len(mempool.Transactions))
	}

	transactions := mempool.GetTransactions()
	if len(transactions) != 2 || !reflect.DeepEqual(transactions, mempool.Transactions) {
		t.Error("GetTransactions() failed, the transactions returned are not correct")
	}

	mempool.Clear()
	if len(mempool.Transactions) != 0 {
		t.Error("Clear() failed, the mempool should be empty")
	}
}
