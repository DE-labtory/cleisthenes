package cleisthenes

import (
	"net"
	"strconv"
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
	return &Member{
		Id: id,
		Addr: Address{
			Ip:   host,
			Port: port,
		},
	}
}

// Address return member's host and port into string
func (m *Member) Address() string {
	return net.JoinHostPort(m.Addr.Ip, strconv.Itoa(int(m.Addr.Port)))
}

// MemberMap manages members information
type MemberMap struct {
	lock    sync.RWMutex
	members map[MemberId]*Member
}

func NewMemberMap() *MemberMap {
	return &MemberMap{
		members: make(map[MemberId]*Member),
		lock:    sync.RWMutex{},
	}
}

// AllMembers returns current members into array format
func (m *MemberMap) Members() []Member {
	m.lock.Lock()
	defer m.lock.Unlock()

	members := make([]Member, 0)
	for _, member := range m.members {
		members = append(members, *member)
	}

	return members
}

func (m *MemberMap) Member(id MemberId) Member {
	m.lock.Lock()
	defer m.lock.Unlock()

	return *m.members[id]
}

func (m *MemberMap) Add(member *Member) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.members[member.Id] = member
}

func (m *MemberMap) Del(id MemberId) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.members, id)
}
