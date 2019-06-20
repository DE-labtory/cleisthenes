package cleisthenes

type Transaction interface{}

type Batch struct {
	// txList is a transaction set of batch which is polled from queue
	txList []Transaction
}

// TxList is a function returns the transaction list on batch
func (batch *Batch) TxList() []Transaction {
	return batch.txList
}
