package merkletree

import (
	"strconv"
	"testing"
)

func setUpMerkleData(n int) []Data {
	var merkleDataList []Data

	for idx := 0; idx < n; idx++ {
		merkleDataList = append(merkleDataList, NewData([]byte(strconv.Itoa(idx))))
	}

	return merkleDataList
}

func TestData_CalculateHash(t *testing.T) {
	data := NewData([]byte("merkle tree's data block"))

	hash, err := data.CalculateHash()
	if err != nil {
		t.Fatalf("error in sha256 hasing : %s", err.Error())
	}

	if len(hash) != 32 {
		t.Fatalf("error in hash size - expected=32, got=%d", len(hash))
	}
}

func TestData_Equals(t *testing.T) {
	data := NewData([]byte("merkle tree's data block"))
	cpyData := NewData([]byte("merkle tree's data block"))

	ok, err := data.Equals(cpyData)
	if err != nil {
		t.Fatalf("error in bytes equals : %s", err.Error())
	}

	if !ok {
		t.Fatalf("not equal data - expected=%x, got=%x", data, cpyData)
	}
}

// merkle tree maintain tree leaf as even number. tree makes new right-leaf
func TestNewMerkleTree(t *testing.T) {
	datas := setUpMerkleData(5)

	_, err := New(datas)
	if err != nil {
		t.Fatalf("error in make merkle tree : %s", err.Error())
	}
}

func TestValidatePath_balance(t *testing.T) {
	dataList := setUpMerkleData(4)

	tree, err := New(dataList)
	if err != nil {
		t.Fatalf("error in NewTree : %s", err.Error())
	}

	rootHash := tree.MerkleRoot()
	rootPath, indexList, err := tree.MerklePath(NewData([]byte(strconv.Itoa(1))))
	if err != nil {
		t.Fatalf("error in GetMerklePath : %s", err.Error())
	}

	if !ValidatePath(NewData([]byte(strconv.Itoa(1))), rootHash, rootPath, indexList) {
		t.Fatalf("error in ValidatePath")
	}
}

func TestValidatePath_unbalance(t *testing.T) {
	dataList := setUpMerkleData(5)

	tree, err := New(dataList)
	if err != nil {
		t.Fatalf("error in NewTree : %s", err.Error())
	}

	rootHash := tree.MerkleRoot()
	rootPath, indexes, err := tree.MerklePath(NewData([]byte(strconv.Itoa(4))))
	if err != nil {
		t.Fatalf("error in GetMerklePath : %s", err.Error())
	}

	if !ValidatePath(NewData([]byte(strconv.Itoa(4))), rootHash, rootPath, indexes) {
		t.Fatalf("error in ValidatePath")
	}
}

func TestOrderOfData(t *testing.T) {
	// test data number 0 to dataNum
	dataNum := 64

	for num := 1; num <= dataNum; num++ {
		datas := setUpMerkleData(num)

		tree, err := New(datas)
		if err != nil {
			t.Fatalf("error in NewTree : %s", err.Error())
		}

		for idx, _ := range datas {
			_, dataList, err := tree.MerklePath(NewData([]byte(strconv.Itoa(idx))))
			if err != nil {
				t.Fatalf("error in GetMerklePath : %s", err.Error())
			}

			order := OrderOfData(dataList)
			if order != idx {
				t.Fatalf("invalid data order - expected : %d, got : %d", idx, order)
			}
		}
	}
}
