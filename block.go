package main


type Block struct {
	Version       int
	PrevBlockHash []byte
	MerkleRoot    []byte
	Hash          []byte
	Time          int64
	Bits          int64
	Nonce         int64
	Transactions  []*Transaction
}
