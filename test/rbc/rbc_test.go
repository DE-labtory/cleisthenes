package rbc

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/cleisthenes/pb"
	"github.com/DE-labtory/cleisthenes/rbc"
	"github.com/DE-labtory/iLogger"
)

type mockHandler struct {
	ServeRequestFunc func(msg cleisthenes.Message)
}

func newMockHandler() *mockHandler {
	return &mockHandler{}
}

func (h *mockHandler) ServeRequest(msg cleisthenes.Message) {
	h.ServeRequestFunc(msg)
}

type mockNode struct {
	n, f int

	// node's address
	address cleisthenes.Address

	rbcMap map[cleisthenes.Address]*rbc.RBC

	server *cleisthenes.GrpcServer

	connPool *cleisthenes.ConnectionPool

	memberMap *cleisthenes.MemberMap
}

func newMockNode(n, f int, addr cleisthenes.Address) *mockNode {
	server := cleisthenes.NewServer(addr)

	connPool := cleisthenes.NewConnectionPool()

	return &mockNode{
		n:        n,
		f:        f,
		rbcMap:   make(map[cleisthenes.Address]*rbc.RBC, 0),
		address:  addr,
		server:   server,
		connPool: connPool,
	}
}

func (m *mockNode) setUpServer() {
	handler := newMockHandler()
	handler.ServeRequestFunc = func(msg cleisthenes.Message) {
		m.serveRequestFunc(msg)
	}

	onConnection := func(conn cleisthenes.Connection) {
		fmt.Println("[server] on connection")
		conn.Handle(handler)
		if err := conn.Start(); err != nil {
			conn.Close()
		}
	}

	m.server.OnConn(onConnection)
}

func (m *mockNode) listen() {
	m.server.Listen()
}

func (m *mockNode) serveRequestFunc(msg cleisthenes.Message) {
	host, port, _ := net.SplitHostPort(msg.GetRbc().Proposer)
	iPort, _ := strconv.Atoi(port)
	proposerAddr := cleisthenes.Address{
		Ip:   host,
		Port: uint16(iPort),
	}
	proposer := m.memberMap.Member(proposerAddr)

	host, port, _ = net.SplitHostPort(msg.Sender)
	iPort, _ = strconv.Atoi(port)
	senderAddr := cleisthenes.Address{
		Ip:   host,
		Port: uint16(iPort),
	}
	sender := m.memberMap.Member(senderAddr)

	r := m.rbcMap[proposer.Address]
	if err := r.HandleMessage(sender, &pb.Message_Rbc{
		Rbc: msg.GetRbc(),
	}); err != nil {
		fmt.Errorf("error in Handlemessage : %s", err.Error())
	}
}

func setMockNodeList(t *testing.T, n, f int) []*mockNode {
	var ip string = "127.0.0.1"
	var port uint16 = 8000

	nodeList := make([]*mockNode, 0)

	for i := 0; i < n; i++ {
		availablePort := GetAvailablePort(port)
		address := cleisthenes.Address{
			Ip:   ip,
			Port: availablePort,
		}
		node := newMockNode(n, f, address)
		node.setUpServer()
		go node.listen()
		nodeList = append(nodeList, node)

		port = availablePort + 1
	}

	for m, mNode := range nodeList {
		// memberMap include proposer node
		memberMap := cleisthenes.NewMemberMap()

		// proposer (owner)
		member := cleisthenes.NewMember(mNode.address.Ip, mNode.address.Port)
		memberMap.Add(member)

		// connPool has only other nodes (except me)
		connPool := cleisthenes.NewConnectionPool()

		//
		// set client and memberMap
		//
		for y, yNode := range nodeList {
			if m != y {
				cli := cleisthenes.NewClient()
				conn, err := cli.Dial(cleisthenes.DialOpts{
					Addr: cleisthenes.Address{
						Ip:   yNode.address.Ip,
						Port: yNode.address.Port,
					},
					Timeout: cleisthenes.DefaultDialTimeout,
				})
				if err != nil {
					t.Fatalf("fail to dial to : %s, with error: %s", yNode.address, err)
				}

				go func() {
					if err := conn.Start(); err != nil {
						conn.Close()
					}
				}()

				connPool.Add(yNode.address, conn)

				member := cleisthenes.NewMember(yNode.address.Ip, yNode.address.Port)
				memberMap.Add(member)
			}
		}

		//
		// set rbc
		//
		owner := memberMap.Member(mNode.address)
		for _, member := range memberMap.Members() {
			r := rbc.New(n, f, owner, member, connPool)
			mNode.rbcMap[member.Address] = r
		}

		mNode.memberMap = memberMap
		mNode.connPool = connPool
	}

	return nodeList
}

func RBCTest(proposer *mockNode, nodeList []*mockNode, data []byte) error {
	iLogger.Infof(nil, "[START] proposer : %s propose data : %s", proposer.address.String(), data)

	proposerMem := proposer.memberMap.Member(proposer.address)

	if err := proposer.rbcMap[proposerMem.Address].HandleInput(data); err != nil {
		return fmt.Errorf("error in MakeRequest : %s", err.Error())
	}

	doneCnt := 0
	for {
		for _, node := range nodeList {
			value := node.rbcMap[proposerMem.Address].Value()
			if value != nil {
				if !bytes.Equal(value, data) {
					return errors.New(fmt.Sprintf("[DONE] node : %s fail to reach an agreement - expected : %s, got : %s\n", node.address.String(), data, value))
				} else {
					iLogger.Infof(nil, "[DONE] node : %s has successfully reached an agreement ! value : %s\n", node.address.String(), value)
				}
				doneCnt++
				break
			}
		}

		if doneCnt == len(nodeList) {
			break
		}
	}

	return nil
}

func Test_RBC_4NODEs(t *testing.T) {
	var n int = 4
	var f int = 1

	nodeList := setMockNodeList(t, n, f)
	for i := 0; i < n; i++ {
		data := fmt.Sprintf("node %d propose consensus", i)
		if err := RBCTest(nodeList[i], nodeList, []byte(data)); err != nil {
			t.Fatalf("test fail : %s", err.Error())
		}
	}
}

func GetAvailablePort(startPort uint16) uint16 {
	portNumber := startPort
	for {
		strPortNumber := strconv.Itoa(int(portNumber))
		lis, err := net.Listen("tcp", "127.0.0.1:"+strPortNumber)

		if err == nil {
			_ = lis.Close()
			return portNumber
		}

		portNumber++
	}
}
