package cleisthenes

type Request interface {
	Recv()
}

type RequestRepository interface {
	Save(addr Address, req Request) error
	Find(addr Address) (Request, error)
	FindAll() []Request
}
