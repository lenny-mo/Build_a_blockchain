package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type Block struct {
	Version       int
	PrevBlockHash []byte
	MerkleRoot    []byte
	Hash          []byte
	Time          int64
	Bits          int64 // 前区块工作量证明的难度目标，这个值是动态调整的，可以保证区块生成的速度大致稳定。
	Nonce         int64
	Transactions  []*Transaction
}

// CreateMerkleRoot creates a merkle root from the transactions
//
// The merkle root is the hash of the root node of the merkle tree
func (b *Block) CreateMerkleRoot() []byte {

	if len(b.Transactions) == 0 {
		return nil
	}

	if len(b.Transactions)%2 != 0 {
		// repeat the last transaction
		b.Transactions = append(b.Transactions, b.Transactions[len(b.Transactions)-1])
	}

	var txHashes [][]byte

	for _, tx := range b.Transactions {
		txHashes = append(txHashes, tx.Hash())
	}

	for len(txHashes) > 1 {
		newTxHashes := [][]byte{}

		for i := 0; i < len(txHashes); i += 2 {
			// concatenate the two hashes and calculate the hash
			hash := sha256.Sum256(append(txHashes[i], txHashes[i+1]...))
			newTxHashes = append(newTxHashes, hash[:])
		}
		txHashes = newTxHashes
	}

	return txHashes[0]

}

// Serialize returns a serialized Block
func (b Block) Serialize() []byte {
	var encoded bytes.Buffer
	err := gob.NewEncoder(&encoded).Encode(b)
	if err != nil {
		panic(err)
	}
	return encoded.Bytes()
}
