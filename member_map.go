package cleisthenes

import (
	"fmt"
	"net"
	"strconv"
	"sync"
)

type Address struct {
	Ip   string
	Port uint16
}

func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.Ip, a.Port)
}

func ToAddress(addrStr string) (Address, error) {
	host, port, err := net.SplitHostPort(addrStr)
	if err != nil {
		return Address{}, err
	}
	portUint, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return Address{}, err
	}
	return Address{
		Ip:   host,
		Port: uint16(portUint),
	}, nil
}

// Member contains Node information who participate in the network
type Member struct {
	Address Address
}

func NewMember(host string, port uint16) *Member {
	return &Member{
		Address: Address{Ip: host, Port: port},
	}
}

// MemberMap manages members information
type MemberMap struct {
	lock    sync.RWMutex
	members map[Address]*Member
}

func NewMemberMap() *MemberMap {
	return &MemberMap{
		members: make(map[Address]*Member),
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

func (m *MemberMap) Member(addr Address) Member {
	m.lock.Lock()
	defer m.lock.Unlock()

	return *m.members[addr]
}

func (m *MemberMap) Add(member *Member) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.members[member.Address] = member
}

func (m *MemberMap) Del(addr Address) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.members, addr)
}
