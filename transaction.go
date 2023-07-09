package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"strings"
)

const (
	coinbasefee = 100
)

type Transaction struct {
	ID  []byte
	In  []TXinput
	Out []TXoutput
}

type TXinput struct {
	TXid      []byte // transaction id
	Voutindex int    // index of the output
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

// CoinbaseTx creates a coinbase transaction
//
// Coinbase 交易是一种特殊的交易，它没有输入，只有输出。
func CoinBaseTx(toAddr string) *Transaction {
	// coinbase transaction has no input, so we use an empty byte slice
	// also, the index of the output is -1 which means it create output without input
	// the signature is nil
	txin := TXinput{[]byte{}, -1, nil}
	// value of coinbase transaction is 100
	txout := TXoutput{coinbasefee, []byte(toAddr)}
	// create a transaction
	tx := Transaction{nil, []TXinput{txin}, []TXoutput{txout}}
	// get the hash of the transaction and set it as the ID
	tx.ID = tx.Hash()

	return &tx
}

func (tx *Transaction) String() string {
	var lines []string

	for i, input := range tx.In {
		lines = append(lines, fmt.Sprintf("Input %d:", i))
		lines = append(lines, fmt.Sprintf("  TXID:      %x", input.TXid))
		lines = append(lines, fmt.Sprintf("  Out:       %d", input.Voutindex))
		lines = append(lines, fmt.Sprintf("  Signature: %x", input.Signature))
	}

	for i, output := range tx.Out {
		lines = append(lines, fmt.Sprintf("Output %d:", i))
		lines = append(lines, fmt.Sprintf("  Value:  %d", output.Value))
		lines = append(lines, fmt.Sprintf("  Script: %x", output.PublickeyHash))
	}

	return strings.Join(lines, "\n")
}
