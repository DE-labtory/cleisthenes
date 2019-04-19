package cleisthenes

import (
	"net"
	"sync"
)

type MemberId = string

// Member contains node information who participate in the network
type Member struct {
	Id   MemberId
	Host net.IP
	Port uint16
}

func NewMember(id string, host string, port int) *Member {
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
