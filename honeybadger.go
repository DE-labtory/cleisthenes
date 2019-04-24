package cleisthenes

type HoneyBadger struct {

	// TODO : HoneyBadger must have ACS

	// TODO : HoneyBadger must have consensused batch

	// que have transaction FIFO memQueue.
	que queue

	// current epoch
	epoch int
}

// Add transaction to FIFO memQueue
func (hb *HoneyBadger) AddTransaction() {
	panic("implement me w/ test case :-)")
}

// Send batch to ACS
func (hb *HoneyBadger) sendBatch() {
	panic("implement me w/ test case :-)")
}

// Create batch polling random transaction in queue
func (hb *HoneyBadger) createBatch() {
	panic("implement me w/ test case :-)")
}

// Define transaction interface.
type Transaction interface{}
