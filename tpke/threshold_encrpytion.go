package tpke

import (
	"github.com/DE-labtory/cleisthenes"
	tpk "github.com/DE-labtory/tpke"
)

type Config struct {
	threshold   int
	participant int
}

type DefaultTpke struct {
	threshold    int
	publicKey    *tpk.PublicKey
	publicKeySet *tpk.PublicKeySet
	secretKey    *tpk.SecretKeyShare
	decShares    map[string]*tpk.DecryptionShare
}

func NewDefaultTpke(th int, skStr cleisthenes.SecretKey, pksStr cleisthenes.PublicKey) (*DefaultTpke, error) {
	sk := tpk.NewSecretKeyFromBytes(skStr)
	sks := tpk.NewSecretKeyShare(sk)

	pks, err := tpk.NewPublicKeySetFromBytes(pksStr)
	if err != nil {
		return nil, err
	}

	return &DefaultTpke{
		threshold:    th,
		publicKeySet: pks,
		publicKey:    pks.PublicKey(),
		secretKey:    sks,
		decShares:    make(map[string]*tpk.DecryptionShare),
	}, nil
}

func (t *DefaultTpke) AcceptDecShare(addr cleisthenes.Address, decShare cleisthenes.DecryptionShare) {
	ds := tpk.NewDecryptionShareFromBytes(decShare)
	t.decShares[addr.String()] = ds
}

func (t *DefaultTpke) ClearDecShare() {
	t.decShares = make(map[string]*tpk.DecryptionShare)
}

// Encrypt encrypts some byte array message.
func (t *DefaultTpke) Encrypt(msg []byte) ([]byte, error) {
	encrypted, err := t.publicKey.Encrypt(msg)
	if err != nil {
		return nil, err
	}
	return encrypted.Serialize(), nil
}

// DecShare makes decryption share using each secret key.
func (t *DefaultTpke) DecShare(ctb cleisthenes.CipherText) cleisthenes.DecryptionShare {
	ct := tpk.NewCipherTextFromBytes(ctb)
	ds := t.secretKey.DecryptShare(ct)
	return ds.Serialize()
}

// Decrypt collects decryption share, and combine it for decryption.
func (t *DefaultTpke) Decrypt(decShares map[string]cleisthenes.DecryptionShare, ctBytes []byte) ([]byte, error) {
	ct := tpk.NewCipherTextFromBytes(ctBytes)
	ds := make(map[string]*tpk.DecryptionShare)
	for id, decShare := range decShares {
		ds[id] = tpk.NewDecryptionShareFromBytes(decShare)
	}
	return t.publicKeySet.DecryptUsingStringMap(ds, ct)
}
