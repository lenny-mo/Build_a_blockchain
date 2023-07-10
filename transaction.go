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
	// input len == 1, lenth of txid == 0, voutindex == -1
	return len(tx.In) == 1 && len(tx.In[0].TXid) == 0 && tx.In[0].Voutindex == -1
}

func (txout *TXoutput) CanBeUnlockedWith(unlockingData string) bool {
	return string(txout.PublickeyHash) == unlockingData
}

func (txin *TXinput) CanUnlockOutputWith(unlockingData string) bool {
	return string(txin.Signature) == unlockingData
}

// NewTXOutput
//
// 遍历所有区块，遍历区块中的所有交易，遍历每个交易中的所有输出，如果输出的公钥哈希值和地址一致，那么这个输出就是我们想要找的
func (bc *Blockchain) FindUTXO(address string) []*Transaction {
	unsepentTXs := []*Transaction{} // 未花费的交易

	spendTxos := make(map[string][]int) // 已花费的交易输出

	bcIterator := bc.Iterator()

	// iterate over all blocks
	for {
		block := bcIterator.Next()

		// iterate over all transactions in one block
		for _, tx := range block.Transactions {
			txID := string(tx.ID)
			
			for outIdx, out := range tx.Out {
				// if the output is already spent, continue
				if spendTxos[txID] != nil {
					
				}
			}
		}

		// 如果区块的前一个区块哈希值为0，说明已经到了创世区块，停止遍历
		if len(block.PrevBlockHash) == 0 {
			break
		}

	}

}
