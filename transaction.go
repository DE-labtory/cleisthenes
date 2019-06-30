package cleisthenes

type Transaction interface{}

// TODO: should be removed, after ACS merged
// TODO: ACS should use BatchMessage instead
type Batch struct {
	// TxList is a transaction set of batch which is polled from queue
	txList []Transaction
}

// TxList is a function returns the transaction list on batch
func (batch *Batch) TxList() []Transaction {
	return batch.txList
}

type Contribution struct {
	TxList []Transaction
}
