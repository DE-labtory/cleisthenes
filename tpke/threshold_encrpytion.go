package tpke

import (
	"github.com/DE-labtory/cleisthenes"
	"github.com/DE-labtory/tpke"
)

type Config struct {
	threshold   int
	participant int
}

type Tpke struct {
	threshold    int
	publicKey    *tpke.PublicKey
	publicKeySet *tpke.PublicKeySet
	secretKey    *tpke.SecretKeyShare
	decShares    map[string]*tpke.DecryptionShare
}

func NewTpke(th int, skStr cleisthenes.SecretKey, pksStr cleisthenes.PublicKey) (*Tpke, error) {
	sk := tpke.NewSecretKeyFromBytes(skStr)
	sks := tpke.NewSecretKeyShare(sk)

	pks, err := tpke.NewPublicKeySetFromBytes(pksStr)
	if err != nil {
		return nil, err
	}

	return &Tpke{
		threshold:    th,
		publicKeySet: pks,
		publicKey:    pks.PublicKey(),
		secretKey:    sks,
		decShares:    make(map[string]*tpke.DecryptionShare),
	}, nil
}

func (t *Tpke) AcceptDecShare(addr cleisthenes.Address, decShare cleisthenes.DecryptionShare) {
	ds := tpke.NewDecryptionShareFromBytes(decShare)
	t.decShares[addr.String()] = ds
}

func (t *Tpke) ClearDecShare() {
	t.decShares = make(map[string]*tpke.DecryptionShare)
}

// Encrypt encrypts some byte array message.
func (t *Tpke) Encrypt(msg []byte) ([]byte, error) {
	encrypted, err := t.publicKey.Encrypt(msg)
	if err != nil {
		return nil, err
	}
	return encrypted.Serialize(), nil
}

// DecShare makes decryption share using each secret key.
func (t *Tpke) DecShare(ctb cleisthenes.CipherText) cleisthenes.DecryptionShare {
	ct := tpke.NewCipherTextFromBytes(ctb)
	ds := t.secretKey.DecryptShare(ct)
	return ds.Serialize()
}

// Decrypt collects decryption share, and combine it for decryption.
func (t *Tpke) Decrypt(decShares map[string]cleisthenes.DecryptionShare, ctBytes []byte) ([]byte, error) {
	ct := tpke.NewCipherTextFromBytes(ctBytes)
	ds := make(map[string]*tpke.DecryptionShare)
	for id, decShare := range decShares {
		ds[id] = tpke.NewDecryptionShareFromBytes(decShare)
	}
	return t.publicKeySet.DecryptUsingStringMap(ds, ct)
}
