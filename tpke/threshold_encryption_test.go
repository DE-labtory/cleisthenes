package tpke

import (
	"testing"

	"github.com/DE-labtory/cleisthenes"

	"github.com/DE-labtory/tpke"
)

type KeySet struct {
	pkSet *tpke.PublicKeySet
	skSet *tpke.SecretKeySet
}

// Setup makes public key set and secret key set.
func Setup(c Config) *KeySet {
	secretKeySet := tpke.RandomSecretKeySet(c.threshold)
	publicKeySet := secretKeySet.PublicKeySet()
	return &KeySet{
		pkSet: publicKeySet,
		skSet: secretKeySet,
	}
}

func TestSetUp(t *testing.T) {
	config := Config{
		threshold:   3,
		participant: 6,
	}

	var privPubKeySet *tpke.PublicKeySet
	var privSecKeySet *tpke.SecretKeySet

	for i := 0; i < 100; i++ {
		keySet := Setup(config)
		if i > 1 {
			if privPubKeySet.Equals(keySet.pkSet) {
				t.Fatalf("KeySet must be different each case.")
			}
			if privSecKeySet.Equals(keySet.skSet) {
				t.Fatalf("a set of key must be different in each case.")
			}
		}
		privPubKeySet = keySet.pkSet
		privSecKeySet = keySet.skSet
	}
}

func TestEncrypt(t *testing.T) {
	config := Config{
		threshold:   3,
		participant: 6,
	}

	msg := []byte("honeyBadger BFT")
	var privCipherText *tpke.CipherText
	keySet := Setup(config)
	for i := 0; i < 100; i++ {
		publicKey := keySet.pkSet.PublicKey()
		ct, err := publicKey.Encrypt(msg)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if i > 1 {
			if privCipherText.Equals(ct) {
				t.Fatalf("a cipherText must be diffrent in each case.")
			}
		}
		privCipherText = ct
	}
}

func TestNewDefaultTpke(t *testing.T) {
	keySet := Setup(Config{
		threshold: 3, participant: 5,
	})

	skSetSerial := keySet.skSet.Serialize()
	skSet, err := tpke.NewSecretKeySetFromBytes(skSetSerial)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !skSet.Equals(keySet.skSet) {
		t.Fatalf("secret key set are different")
	}

	pkSetSerial := keySet.pkSet.Serialize()
	pkSet, err := tpke.NewPublicKeySetFromBytes(pkSetSerial)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !pkSet.Equals(keySet.pkSet) {
		t.Fatalf("public key set are different")
	}

	tpke0, err := NewDefaultTpke(5, skSet.KeyShareUsingString("0").Serialize(), pkSetSerial)
	if err != nil {
		t.Fatalf("%v", err)
	}
	tpke1, err := NewDefaultTpke(5, skSet.KeyShareUsingString("1").Serialize(), pkSetSerial)
	tpke2, err := NewDefaultTpke(5, skSet.KeyShareUsingString("2").Serialize(), pkSetSerial)
	tpke3, err := NewDefaultTpke(5, skSet.KeyShareUsingString("3").Serialize(), pkSetSerial)
	tpke4, err := NewDefaultTpke(5, skSet.KeyShareUsingString("4").Serialize(), pkSetSerial)

	msg := []byte("DE-labtory")
	cipher0, err := tpke0.Encrypt(msg)

	if err != nil {
		t.Fatalf("encyption failed : %v", err)
	}

	decShareMap := make(map[string]cleisthenes.DecryptionShare)

	decShare0 := tpke0.DecShare(cipher0)
	decShareMap["0"] = decShare0
	_, err = tpke0.Decrypt(decShareMap, cipher0)
	t.Logf("decrypt must be failed. %v", err)

	decShare1 := tpke1.DecShare(cipher0)
	decShareMap["1"] = decShare1
	_, err = tpke0.Decrypt(decShareMap, cipher0)
	t.Logf("decrypt must be failed. %v", err)

	decShare2 := tpke2.DecShare(cipher0)
	decShareMap["2"] = decShare2
	_, err = tpke0.Decrypt(decShareMap, cipher0)
	t.Logf("decrypt must be failed. %v", err)

	decShare3 := tpke3.DecShare(cipher0)
	decShareMap["3"] = decShare3
	result, _ := tpke0.Decrypt(decShareMap, cipher0)
	t.Logf("result : %v", string(result))

	decShare4 := tpke4.DecShare(cipher0)
	decShareMap["4"] = decShare4
	result, _ = tpke0.Decrypt(decShareMap, cipher0)
	t.Logf("result : %v", string(result))
}
