package cleisthenes

type Request interface {
	Recv()
}

type RequestRepository interface {
	Add(connId ConnId, req Request) error
	Find(connId ConnId) Request
	FindAll() []Request
}
