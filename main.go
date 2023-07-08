package main

import "fmt"

func main() {
	block := Block{
		Version:       1,
		PrevBlockHash: []byte(""),
		MerkleRoot:    []byte(""),
		Time:          0,
		Bits:          0,
		Nonce:         0,
		Transactions:  []*Transaction{},
	}

	pow := NewPOW(&block)

	nonce, _ := pow.Run()

	block.Nonce = nonce

	rs := pow.Validate()

	fmt.Println(rs)
}
