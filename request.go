package cleisthenes

type Request interface {
	Recv()
}

type RequestRepository interface {
	Save(connId ConnId, req Request) error
	Find(connId ConnId) (Request, error)
	FindAll() []Request
}

type IncomingRequestRepository interface {
	Save(epoch int, connId ConnId, req Request) error
	Find(epoch int, connId ConnId) Request
	FindByEpoch(epoch int) []Request
}
