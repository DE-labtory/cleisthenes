package cleisthenes

type Request interface {
	Recv()
}

type RequestRepository interface {
	Save(addr Address, req Request) error
	Find(addr Address) (Request, error)
	FindAll() []Request
}

type IncomingRequestRepository interface {
	Save(epoch int, addr Address, req Request) error
	Find(epoch int, addr Address) Request
	FindByEpoch(epoch int) []Request
}
