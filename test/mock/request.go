package mock

import "github.com/DE-labtory/cleisthenes"

type RequestRepository struct {
	ReqMap      map[cleisthenes.ConnId]cleisthenes.Request
	SaveFunc    func(connId cleisthenes.ConnId, req cleisthenes.Request) error
	FindFunc    func(connId cleisthenes.ConnId) (cleisthenes.Request, error)
	FindAllFunc func() []cleisthenes.Request
}

func (r *RequestRepository) Save(connId cleisthenes.ConnId, req cleisthenes.Request) error {
	return r.SaveFunc(connId, req)
}
func (r *RequestRepository) Find(connId cleisthenes.ConnId) (cleisthenes.Request, error) {
	return r.FindFunc(connId)
}
func (r *RequestRepository) FindAll() []cleisthenes.Request {
	return r.FindAllFunc()
}
