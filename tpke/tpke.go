package tpke

import (
	"encoding/json"
	"github.com/DE-labtory/cleisthenes"
)

type MockTpke struct{}

func (t *MockTpke) Encrypt(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (t *MockTpke) Decrypt(enc []byte) ([]byte, error) {
	return enc, nil
}

func (t *MockTpke) DecShare(ctBytes []byte) cleisthenes.DecryptionShare {
	return [96]byte{}
}

func (t *MockTpke) AcceptDecShare(addr cleisthenes.Address, decShare cleisthenes.DecryptionShare) {

}

func (t *MockTpke) ClearDecShare() {}
