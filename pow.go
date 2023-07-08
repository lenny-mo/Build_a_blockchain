package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math/big"
)

const (
	targetBits       = 16        // 挖矿难度值，这里是16，表示hash值的前16位必须是0
	maxNonce   int64 = 1<<63 - 1 // 2^63 - 1
)

type POW struct {
	block  *Block
	Target *big.Int
}

// NewPOW creates a new POW
func NewPOW(b *Block) *POW {

	target := big.NewInt(1)
	// left shift 256 - targetBits, which
	target.Lsh(target, uint(256-targetBits))

	pow := &POW{b, target}
	return pow
}

// ConvertData2Bytes convert the block into a byte array
//
// 把区块转换成字节数组
func (pow *POW) ConvertData2Bytes(nonce int64) []byte {

	data := bytes.Join(
		[][]byte{
			Uint64ToBytesBigEndian(uint64(pow.block.Version)),
			pow.block.PrevBlockHash,
			pow.block.MerkleRoot,
			Uint64ToBytesBigEndian(uint64(pow.block.Time)),
			Uint64ToBytesBigEndian(uint64(pow.block.Bits)),
			Uint64ToBytesBigEndian(uint64(nonce)),
		},
		[]byte{}, // concat without separator
	)

	return data
}

// Run performs the POW to find a nonce that satisfies the target
// 通过不断的尝试，找到一个合适的nonce，使得区块的hash值小于目标值
//
// return value: nonce, target hash
func (pow *POW) Run() (int64, []byte) {
	var nonce int64
	var currentHash big.Int
	var firstHash, secondHash [32]byte

	for nonce < maxNonce {
		// serialize the block
		powData := pow.ConvertData2Bytes(nonce)
		// double sha256 to enhance the security
		firstHash = sha256.Sum256(powData)
		secondHash = sha256.Sum256(firstHash[:])

		fmt.Printf("\r%x\n", secondHash)
		// convert the hash to a big integer
		currentHash.SetBytes(secondHash[:])

		// if currentHash < target, we found the nonce
		if currentHash.Cmp(pow.Target) == -1 {
			// found the nonce
			break
		} else {
			nonce++
		}
	}

	// no nonce found
	return nonce, secondHash[:]
}

// Validate validates if the nonce is valid
func (pow *POW) Validate() bool {
	var hashInt big.Int

	data := pow.ConvertData2Bytes(pow.block.Nonce)
	firstHash := sha256.Sum256(data)
	secondHash := sha256.Sum256(firstHash[:])
	hashInt.SetBytes(secondHash[:])

	return hashInt.Cmp(pow.Target) == -1
}
