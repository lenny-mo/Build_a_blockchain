package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
)

type Transaction struct {
	ID  []byte
	In  []TXinput
	Out []TXoutput
}

type TXinput struct {
	TXid      []byte
	Voutindex int
	Signature []byte
}

type TXoutput struct {
	Value         int
	PublickeyHash []byte
}

// Serialize returns a serialized Transaction
func (tx Transaction) Serialize() []byte {
	var encoded bytes.Buffer
	err := gob.NewEncoder(&encoded).Encode(tx)

	if err != nil {
		panic(err)
	}

	return encoded.Bytes()
}
    
// Hash returns the hash of the Transaction
func (tx Transaction) Hash() []byte {

	hash := sha256.Sum256(tx.Serialize())

	// return a slice of the hash
	return hash[:]
}
