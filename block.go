package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"strings"
	"time"
)

type Block struct {
	Version       int
	PrevBlockHash []byte
	MerkleRoot    []byte
	Hash          []byte
	Time          int64 // 这个数字表示从UNIX纪元开始到现在的秒数
	Bits          int64 // 前区块工作量证明的难度目标，这个值是动态调整的，可以保证区块生成的速度大致稳定。
	Nonce         int64 // 用于工作量证明算法的计数器
	Transactions  []*Transaction
}

// CreateMerkleRoot creates a merkle root from the transactions
//
// The merkle root is the hash of the root node of the merkle tree
func (b *Block) CreateMerkleRoot() []byte {

	// if there are no transactions, merkle root is nil
	if len(b.Transactions) == 0 {
		return nil
	}

	// number of transactions is odd, repeat the last transaction
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

// Deserialize returns a deserialized Block pointer
func Deserialize(d []byte) *Block {
	var block Block
	err := gob.NewDecoder(bytes.NewReader(d)).Decode(&block)
	if err != nil {
		panic(err)
	}
	return &block
}

func (b *Block) String() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Version: %d", b.Version))
	lines = append(lines, fmt.Sprintf("Prev. block: %x", b.PrevBlockHash))
	lines = append(lines, fmt.Sprintf("Merkle root: %x", b.MerkleRoot))
	lines = append(lines, fmt.Sprintf("Timestamp: %d", b.Time))
	lines = append(lines, fmt.Sprintf("Bits: %d", b.Bits))
	lines = append(lines, fmt.Sprintf("Nonce: %d", b.Nonce))

	return strings.Join(lines, "\n")
}

// GenesisBlock creates and returns genesis block
//
// 创世纪区块是区块链中的第一个区块，它是在区块链系统启动时创建的，而不是像其他区块一样通过工作量证明算法创建的。
func GenesisBlock() *Block {
	coinbaseTx := CoinBaseTx("Genesis Block")
	block := Block{
		1,                          // Version= 1
		[]byte{},                   // PrevBlockHash= nil
		nil,                        // MerkleRoot= nil
		nil,                        // Hash= nil
		time.Now().Unix(),          // Time= 0
		0,                          // Bits= 0
		0,                          // Nonce= 0
		[]*Transaction{coinbaseTx}} // Transaction list

	return &block
}

// NewBlock creates and returns Block
//
// 该函数接收一个前区块的哈希值和一个交易列表，然后创建一个新的区块，返回该区块的指针。
func NewBlock(prevBlockHash []byte, transactions []*Transaction) *Block {
	block := &Block{
		2,                 // Version= 2
		prevBlockHash,     // PrevBlockHash= prevBlockHash
		nil,               // MerkleRoot= nil
		nil,               // Hash= nil
		time.Now().Unix(), // Time= 0
		0,                 // Bits= 0
		0,                 // Nonce= 0
		transactions}      // Transaction list

	pow := NewPOW(block)
	// calculate the nonce and hash
	nonce, hash := pow.Run()
	block.Nonce, block.Hash = nonce, hash[:]
	return block
}
