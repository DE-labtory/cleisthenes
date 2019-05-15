package cleisthenes

import (
	"crypto/sha256"

	"github.com/Nik-U/pbc"
	"github.com/SSSaaS/sssa-golang"
)

type Batch struct {
}

var Enc = func() []byte {
	//msg2 := []byte(msg)

	return nil
}
var DecShare = func() *KeyList {
	return nil
}
var Decrypt = func() (*Batch, error) {
	return nil, nil
}

///// shamir secret share
func shamir(min int, shares int, raw string) ([]string, error) {
	return sssa.Create(min, shares, raw)
}

func combine(shares []string) (string, error) {
	return sssa.Combine(shares)
}

////// boneh-franklin

type Ibe struct {
	g1 *pbc.Element
	g2 *pbc.Element
}

func SetUp() *Ibe {
	params := pbc.GenerateA(160, 512)
	pairing := params.NewPairing()
	g1 := pairing.NewG1().Rand()
	g2 := pairing.NewG2().Rand()


	return &Ibe {
		g1: g1,
		g2: g2,
	}
}

// BLS signing
type MessageData struct {
	message   string
	signature []byte
}

type SharedData struct {
	sharedParams string
	sharedG      []byte
	pubKey       []byte
}

func BlsGenerate(rbits, qbits uint32) (*SharedData, []byte) {
	// params := pbc.GenerateA(160, 512)
	params := pbc.GenerateA(rbits, qbits)
	pairing := params.NewPairing()
	g := pairing.NewG2().Rand()
	privKey := pairing.NewZr().Rand()
	pubKey := pairing.NewG2().PowZn(g, privKey)

	return &SharedData{
		sharedParams: params.String(),
		sharedG:      g.Bytes(),
		pubKey:       pubKey.Bytes(),
	}, privKey.Bytes()
}

func BlsSigning(sharedData *SharedData, bPrivKey []byte, msg string) *MessageData {
	pairing, _ := pbc.NewPairingFromString(sharedData.sharedParams)
	privKey := pairing.NewZr().SetBytes(bPrivKey)
	h := pairing.NewG1().SetFromStringHash(msg, sha256.New())
	signature := pairing.NewG2().PowZn(h, privKey)

	return &MessageData{
		message:   msg,
		signature: signature.Bytes(),
	}
}

func BlsVerifying(sharedData *SharedData, msgData *MessageData) bool {
	pairing, _ := pbc.NewPairingFromString(sharedData.sharedParams)
	g := pairing.NewG2().SetBytes(sharedData.sharedG)

	signature := pairing.NewG1().SetBytes(msgData.signature)
	pk := pairing.NewG2().SetBytes(sharedData.pubKey)

	h := pairing.NewG1().SetFromStringHash(msgData.message, sha256.New())
	temp1 := pairing.NewGT().Pair(h, pk)
	temp2 := pairing.NewGT().Pair(signature, g)
	return temp1.Equals(temp2)
}
