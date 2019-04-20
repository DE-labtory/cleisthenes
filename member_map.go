package cleisthenes

import (
	"sync"
)

type MemberId = string

type Address struct {
	Ip   string
	Port uint16
}

// Member contains node information who participate in the network
type Member struct {
	Id   MemberId
	Addr Address
}

func NewMember(id string, host string, port uint16) *Member {
	panic("implement me w/ test case :-)")
}

// Address return member's host and port into string
func (m *Member) Address() string {
	panic("implement me w/ test case :-)")
}

// MemberMap manages members information
type MemberMap struct {
	lock    sync.RWMutex
	members map[MemberId]*Member
}

func NewMemberMap() *MemberMap {
	panic("implement me w/ test case :-)")
}

// AllMembers returns current members into array format
func (m *MemberMap) AllMembers() []Member {
	panic("implement me w/ test case :-)")
}

func (m *MemberMap) Member(id MemberId) Member {
	panic("implement me w/ test case :-)")
}

func (m *MemberMap) Add(member *Member) {
	panic("implement me w/ test case :-)")
}

func (m *MemberMap) Del(id MemberId) {
	panic("implement me w/ test case :-)")
}
