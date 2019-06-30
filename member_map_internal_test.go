/*
 * Copyright 2019 DE-labtory
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cleisthenes

import (
	"reflect"
	"testing"
)

func TestMemberMap_Add(t *testing.T) {
	addr := Address{
		Ip:   "localhost",
		Port: 8000,
	}

	memberMap := NewMemberMap()
	memberMap.Add(newDummyMember(addr))

	if !reflect.DeepEqual(*memberMap.members[addr], *newDummyMember(addr)) {
		t.Fatalf("two member are not equal. got=%v+x, expected=%v+x", memberMap.members[addr], *newDummyMember(addr))
	}
}

func TestMemberMap_Del(t *testing.T) {
	addr := Address{
		Ip:   "localhost",
		Port: 8000,
	}

	memberMap := NewMemberMap()
	memberMap.members[addr] = newDummyMember(addr)

	memberMap.Del(addr)

	_, ok := memberMap.members[addr]
	if ok {
		t.Fatalf("deleted member exists")
	}

	if len(memberMap.members) != 0 {
		t.Fatalf("size of membermap is not 0. got=%d", len(memberMap.members))
	}
}

func TestMemberMap_Member(t *testing.T) {
	addr := Address{
		Ip:   "localhost",
		Port: 8000,
	}

	memberMap := NewMemberMap()
	memberMap.members[addr] = newDummyMember(addr)

	member, ok := memberMap.Member(addr)
	if !ok {
		t.Fatalf("expect member to exist")
	}
	if !reflect.DeepEqual(member, *newDummyMember(addr)) {
		t.Fatalf("two member are not equal. got=%v+x, expected=%v+x", memberMap.members[addr], *newDummyMember(addr))
	}
}

func TestMemberMap_Members(t *testing.T) {
	addr := Address{
		Ip:   "localhost",
		Port: 8000,
	}
	addr2 := Address{
		Ip:   "localhost",
		Port: 8001,
	}

	memberMap := NewMemberMap()
	memberMap.members[addr] = newDummyMember(addr)
	memberMap.members[addr2] = newDummyMember(addr2)

	if len(memberMap.Members()) != 2 {
		t.Fatalf("size of membermap is not 2. got=%d", len(memberMap.Members()))
	}

	for _, member := range memberMap.Members() {
		if !reflect.DeepEqual(*memberMap.members[member.Address], member) {
			t.Fatalf("two member are not equal. got=%v+x, expected=%v+x", *memberMap.members[member.Address], member)
		}
	}
}

func newDummyMember(addr Address) *Member {
	return &Member{
		Address: addr,
	}
}
