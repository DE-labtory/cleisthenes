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
	memberMap := NewMemberMap()

	memberMap.Add(newDummyMember("memberId"))

	if !reflect.DeepEqual(*memberMap.members["memberId"], *newDummyMember("memberId")) {
		t.Fatalf("two member are not equal. got=%v+x, expected=%v+x", memberMap.members["memberId"], *newDummyMember("memberId"))
	}
}

func TestMemberMap_Del(t *testing.T) {
	memberMap := NewMemberMap()
	memberMap.members["memberId"] = newDummyMember("memberId")

	memberMap.Del("memberId")

	_, ok := memberMap.members["memberId"]
	if ok {
		t.Fatalf("deleted member exists")
	}

	if len(memberMap.members) != 0 {
		t.Fatalf("size of membermap is not 0. got=%d", len(memberMap.members))
	}
}

func TestMemberMap_Member(t *testing.T) {
	memberMap := NewMemberMap()
	memberMap.members["memberId"] = newDummyMember("memberId")

	member := memberMap.Member("memberId")
	if !reflect.DeepEqual(member, *newDummyMember("memberId")) {
		t.Fatalf("two member are not equal. got=%v+x, expected=%v+x", memberMap.members["memberId"], *newDummyMember("memberId"))
	}
}

func TestMemberMap_Members(t *testing.T) {
	memberMap := NewMemberMap()
	memberMap.members["memberId1"] = newDummyMember("memberId1")
	memberMap.members["memberId2"] = newDummyMember("memberId2")

	if len(memberMap.Members()) != 2 {
		t.Fatalf("size of membermap is not 2. got=%d", len(memberMap.Members()))
	}

	for _, member := range memberMap.Members() {
		if !reflect.DeepEqual(*memberMap.members[member.Id], member) {
			t.Fatalf("two member are not equal. got=%v+x, expected=%v+x", *memberMap.members[member.Id], member)
		}
	}
}

func newDummyMember(memberId string) *Member {
	return &Member{
		Id: memberId,
		Addr: Address{
			Ip:   "localhost",
			Port: 8080,
		},
	}
}
