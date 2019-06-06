package merkletree

import (
	"bytes"
	"crypto/sha256"
	"math"

	"github.com/cbergoon/merkletree"
)

type MerkleData = merkletree.Content
type RootPath [][]byte
type RootHash []byte

type Wrapper struct {
	tree merkletree.MerkleTree
}

func (w *Wrapper) MerkleRoot() RootHash {
	return w.tree.MerkleRoot()
}

func (w *Wrapper) MerklePath(data Data) (RootPath, []int64, error) {
	return w.tree.GetMerklePath(data.Content)
}

func New(dataList []Data) (*Wrapper, error) {
	var merkleDataList []MerkleData

	for _, data := range dataList {
		merkleDataList = append(merkleDataList, data.Content)
	}

	t, err := merkletree.NewTree(merkleDataList)
	if err != nil {
		return nil, err
	}

	return &Wrapper{*t}, nil
}

type Data struct {
	Content content
}

func NewData(data []byte) Data {
	return Data{Content: content(data)}
}

func (d Data) CalculateHash() ([]byte, error) {
	return d.Content.CalculateHash()
}

func (d Data) Equals(a Data) (bool, error) {
	return d.Content.Equals(a.Content)
}

func (d Data) Bytes() []byte {
	return d.Content
}

type content []byte

func (c content) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(c)); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

func (c content) Equals(other merkletree.Content) (bool, error) {
	return bytes.Equal(c, other.(content)), nil
}

func ValidatePath(data Data, rootHash RootHash, rootPath RootPath, indexList []int64) bool {
	leaf := make(content, 64)
	branch, err := data.CalculateHash()
	if err != nil {
		return false
	}

	for i, path := range rootPath {

		// when it is left leaf
		if indexList[i] == 0 {
			copy(leaf[:32], path)
			copy(leaf[32:], branch)
			//copy(branch[len(branch):len(branch) + len(path)], path)
			branch, err = leaf.CalculateHash()
		}

		// when it is right leaf
		if indexList[i] == 1 {
			copy(leaf[:32], branch)
			copy(leaf[32:], path)
			//copy(path[len(path):len(path) + len(branch)], branch)
			//copy(branch, path)
			branch, err = leaf.CalculateHash()
		}

		if err != nil {
			return false
		}
	}

	if bytes.Equal(rootHash, branch) {
		return true
	}

	return false
}

// OrderOfData return data's order as slice's order (slice's first index is 0)
// merkletree's MerklePath return rootPath, indexes.
// indexes is indicate whether the leaf is left of right.
// so, you can find the order of the data in indexes.
func OrderOfData(indexList []int64) int {
	order := 0
	for i, idx := range indexList {
		num := int(math.Pow(2, float64(i+1)))
		if idx == 0 {
			order += num / 2
		}
	}

	return order
}
