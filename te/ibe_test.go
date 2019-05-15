package te

import (
	"reflect"
	"testing"
)

func TestIbe_SetUp(t *testing.T) {
	_, err := SetUp()
	if err != nil {
		t.Fatalf("test failed. err = %s", err.Error())
	}
}

func TestIbe_MakePrivateKey(t *testing.T) {
	m, _ := SetUp()
	_, err := MakePrivateKey(m, "test")
	if err != nil {
		t.Fatalf("test failed. %s", err.Error())
	}
}

func TestIbe_Encrypt(t *testing.T) {
	m, _ := SetUp()
	msg := "testMessage"
	enc := Encrypt(m, "testId", []byte(msg))
	if enc == nil || reflect.DeepEqual(msg, string(enc)) {
		t.Fatalf("test encryption failed")
	}
}

func TestIbe_Decrypt(t *testing.T) {
	m, _ := SetUp()
	id := "alice"
	msg := []byte("testMessage")
	enc := Encrypt(m, id, msg)
	pKey, _ := MakePrivateKey(m, id)
	dec := Decrypt(m, pKey, enc)
	if !reflect.DeepEqual(msg, dec) {
		t.Fatalf("test decryption failed.")
	}
}