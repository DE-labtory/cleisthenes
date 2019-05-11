package mock

import "github.com/DE-labtory/cleisthenes"

type RequestRepository struct {
	ReqMap      map[cleisthenes.Address]cleisthenes.Request
	SaveFunc    func(addr cleisthenes.Address, req cleisthenes.Request) error
	FindFunc    func(addr cleisthenes.Address) (cleisthenes.Request, error)
	FindAllFunc func() []cleisthenes.Request
}

func (r *RequestRepository) Save(addr cleisthenes.Address, req cleisthenes.Request) error {
	return r.SaveFunc(addr, req)
}
func (r *RequestRepository) Find(addr cleisthenes.Address) (cleisthenes.Request, error) {
	return r.FindFunc(addr)
}
func (r *RequestRepository) FindAll() []cleisthenes.Request {
	return r.FindAllFunc()
}
