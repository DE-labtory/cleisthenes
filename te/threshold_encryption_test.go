package cleisthenes

import (
	"testing"
)

func TestTE_enc(t *testing.T) {

}

func TestTE_decShare(t *testing.T) {

}

func TestTE_decrypt(t *testing.T) {

}

/// test bls sign
func TestTE_BLS_signing(t *testing.T) {
	sharedData, privKey := BlsGenerate(160, 512)
	msgData := BlsSigning(sharedData, privKey, "hi!")
	result := BlsVerifying(sharedData, msgData)
	t.Logf("result : %v", result)
}

func TestTE_Ibe(t *testing.T) {
	ibe := SetUp()
	t.Logf("%v", ibe.g1)
	t.Logf("%v", ibe.g2)
}
