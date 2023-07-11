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
	ID  []byte     // 交易的哈希值
	In  []TXinput  // 交易的所有输入。每一个 TXinput 都包含一个引用到过去交易的未花费输出UTXO，这表示你想要花费这些比特币。
	Out []TXoutput // 交易的所有输出。每一个 TXoutput 都定义了一个新的比特币所有者和他们将获得的比特币数量。
}

type TXinput struct {
	TXid      []byte //	我们想要引用哪个过去的交易
	Voutindex int    // 过去交易的哪一个输出。比如说，如果一个过去的交易有多个输出，我们可以用 Voutindex 来确定我们想要引用哪一个
	Signature []byte // 交易输入的签名。这个签名是由交易的发送者生成的，用来证明他们有权利花费这个交易输入所引用的UTXO
}

type TXoutput struct {
	Value         int    // 该交易输出中包含的比特币数量
	PublickeyHash []byte // 公钥哈希值用来标识比特币的新所有者
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
// Coinbase 交易是一种特殊的交易，它没有任何输入，只有一个输出，toaddr是收款地址
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

// String returns a human-readable representation of a transaction
//
// 交易的字符串表示
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

// IsCoinbase checks whether the transaction is coinbase
//
// 判断交易是否是coinbase交易
func (tx Transaction) IsCoinbase() bool {
	// input len == 1, lenth of txid == 0(no tx), voutindex == -1(no output)
	return len(tx.In) == 1 && len(tx.In[0].TXid) == 0 && tx.In[0].Voutindex == -1
}

func (txout *TXoutput) CanBeUnlockedWith(unlockingData string) bool {
	return string(txout.PublickeyHash) == unlockingData
}

func (txin *TXinput) CanUnlockOutputWith(unlockingData string) bool {
	return string(txin.Signature) == unlockingData
}

// CreateTransaction creates a new transaction
//
// 创建一个新的交易
func CreateTransaction(fromAddr, toAddr string, amount int, blockchain *Blockchain) *Transaction {
	inputs := []TXinput{}
	outputs := []TXoutput{}

	actualBalance, tx_index := blockchain.FindSpendableOutputs(fromAddr, amount)

	// 如果余额不足，返回 nil
	if actualBalance < amount {
		fmt.Println("ERROR: Not enough funds")
		return nil
	}

	// iterate over the tx_index mapping, which contains the unspent output index
	for txid, outputList := range tx_index {
		txidStr := string(txid)

		// iterate over the outputList, which contains the unspent output index
		for _, outputIndex := range outputList {
			// create a new input
			input := TXinput{[]byte(txidStr), outputIndex, []byte(fromAddr)}
			// append the input to the inputs
			inputs = append(inputs, input)
		}
	}

	// create a new output for the receiver
	output := TXoutput{amount, []byte(toAddr)}
	// append the output to the outputs
	outputs = append(outputs, output)

	// if the actualBalance is greater than the amount,
	// we need to send the change back to the sender
	if actualBalance > amount {
		// create a new output for the sender
		output := TXoutput{actualBalance - amount, []byte(fromAddr)}
		// append the output to the outputs
		outputs = append(outputs, output)
	}

	// create a new transaction
	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()

	return &tx
}
