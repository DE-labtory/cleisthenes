package cleisthenes

type SecretKey [32]byte
type PublicKey []byte
type DecryptionShare [96]byte
type CipherText []byte

type Tpke interface {
	Encrypt(msg interface{}) ([]byte, error)
	DecShare(ctBytes []byte) DecryptionShare
	Decrypt(decShares map[string]DecryptionShare, ctBytes CipherText) ([]byte, error)
	AcceptDecShare(addr Address, decShare DecryptionShare)
	ClearDecShare()
}
