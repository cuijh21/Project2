package main

// Mempool represents a memory pool for transactions.
type Mempool struct {
	Transactions []*Transaction // A slice of pointers to transactions
}

// NewMempool creates and returns a new Mempool instance.
func NewMempool() *Mempool {
	return &Mempool{Transactions: []*Transaction{}}
}

// AddTransaction adds a transaction to the Mempool.
func (m *Mempool) AddTransaction(tx *Transaction) {
	m.Transactions = append(m.Transactions, tx)
}

// GetTransactions returns all transactions in the Mempool.
func (m *Mempool) GetTransactions() []*Transaction {
	return m.Transactions
}

// Clear empties all transactions from the Mempool.
func (m *Mempool) Clear() {
	m.Transactions = []*Transaction{}
}
