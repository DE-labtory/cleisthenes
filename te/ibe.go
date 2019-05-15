package te

import "v.io/x/lib/ibe"

func SetUp() (ibe.Master, error) {
	return ibe.SetupBB2()
}

func MakeMasterKey(master ibe.Master) ([]byte, error){
	return ibe.MarshalMasterKey(master)
}

func MakePrivateKey(master ibe.Master, id string) (ibe.PrivateKey, error) {
	return master.Extract(id)
}

func Encrypt(master ibe.Master, id string, msg []byte) []byte {
	param := master.Params()
	buf := make([]byte, len(msg) + param.CiphertextOverhead())
	param.Encrypt(id, msg, buf)
	return buf
}

func Decrypt(master ibe.Master, pKey ibe.PrivateKey, eMsg []byte) []byte {
	buf := make([]byte, len(eMsg) - master.Params().CiphertextOverhead())
	pKey.Decrypt(eMsg, buf)
	return buf
}