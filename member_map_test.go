package cleisthenes_test

import (
	"testing"

	"github.com/DE-labtory/cleisthenes"
)

func TestMember_Address(t *testing.T) {
	member := cleisthenes.Member{
		Id: "memberId",
		Addr: cleisthenes.Address{
			Ip:   "localhost",
			Port: 8080,
		},
	}
	if member.Address() != "localhost:8080" {
		t.Fatalf("member address are not localhost:8080. got=%s", member.Address())
	}
}
