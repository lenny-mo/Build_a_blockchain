package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

const (
	COINBASEFEE = 100 // coinbase交易给矿工的奖励
)

type Transaction struct {
	ID  []byte     // 交易的哈希值
	In  []TXinput  // 交易的所有输入。每一个 TXinput 都包含一个引用到过去交易的未花费输出UTXO，这表示你想要花费这些比特币。
	Out []TXoutput // 交易的所有输出。每一个 TXoutput 都定义了一个新的比特币所有者和他们将获得的比特币数量。
}

type TXinput struct {
	TXid      []byte //	我们想要引用哪个过去的交易
	Voutindex int    // 使用上一个交易中的第几个输出。比如说，如果一个过去的交易有多个输出，我们可以用 Voutindex 来确定我们想要引用哪一个
	Signature []byte // 交易输入的签名。这个签名是由交易的发送者生成的，用来证明他们有权利花费这个交易输入所引用的UTXO
	Pubkey    []byte // 公钥
}

type TXoutput struct {
	Value         int    // 该交易输出中包含的比特币数量
	PublickeyHash []byte // 公钥哈希值用来标识比特币的新所有者
}

type TXoutputSlice []TXoutput

// Serialize returns a serialized TXinput
//
// 序列化交易输入
func (outs *TXoutputSlice) Serialize() []byte {
	var buffer bytes.Buffer // 编码后的字节流

	encoder := gob.NewEncoder(&buffer)

	err := encoder.Encode(outs)
	if err != nil {
		panic(err)
	}

	return buffer.Bytes()
}

// DeserializeOutputSlice deserializes a TXoutputSlice from a byte slice
//
// 根据字节流反序列化交易输出切片
func DeserializeOutputSlice(data []byte) TXoutputSlice {
	var outputs TXoutputSlice
	decoder := gob.NewDecoder(bytes.NewReader(data))

	err := decoder.Decode(&outputs)
	if err != nil {
		panic(err)
	}

	return outputs
}

// Lock signs the output
//
// 交易输出锁定, 根据收款人的地址计算出公钥哈希并且赋值给交易输出的公钥哈希字段
// 只有拥有相应私钥的用户（即接收者）才能解锁（也就是花费）这个交易输出。
func (out *TXoutput) LockAddress(address string) {
	decodedAddr, err := Base58Decode([]byte(address))
	if err != nil {
		fmt.Println("decode address error", err)
		panic(err)
	}

	pubkeyHash := decodedAddr[1 : len(decodedAddr)-4]

	out.PublickeyHash = pubkeyHash
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
	// pubkey is also nil
	txin := TXinput{[]byte{}, -1, nil, []byte{}}
	// value of coinbase transaction is 100
	txout := TXoutput{COINBASEFEE, AddressToPubkeyHash(toAddr)}
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
// 判断交易是否是coinbase交易, 返回false表示不是coinbase交易
func (tx Transaction) IsCoinbase() bool {
	// input len == 1, lenth of txid == 0(no tx), voutindex == -1(no output)
	return len(tx.In) == 1 && len(tx.In[0].TXid) == 0 && tx.In[0].Voutindex == -1
}

func (txout *TXoutput) CanBeUnlockedWith(pubkeyHash []byte) bool {
	return bytes.Equal(txout.PublickeyHash, pubkeyHash)
}

// CanUnlockOutputWith checks whether the given pubkeyhash can unlock the output
//
// 检查公钥哈希是否能够解锁交易输出
func (txin *TXinput) CanUnlockOutputWith(pubkeyHash []byte) bool {
	localPubkeyHash := PublickeyHash(txin.Pubkey)
	return bytes.Equal(localPubkeyHash, pubkeyHash)
}

// CreateTransaction creates a new transaction
//
// 创建一个新的交易
func CreateTransaction(fromAddr, toAddr string, amount int, blockchain *Blockchain) *Transaction {
	inputs := []TXinput{}
	outputs := []TXoutput{}

	// create a wallet to get pubkey
	wallets := CreateWallets()
	senderKeyPair := wallets.GetWallet(fromAddr) // 根据地址获取公私钥对

	// 获取fromAddr的所有未花费输出的总额和索引
	actualBalance, tx_index := blockchain.FindSpendableOutputs(PublickeyHash(senderKeyPair.PublicKey), amount)

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
			input := TXinput{[]byte(txidStr), outputIndex, nil, senderKeyPair.PublicKey}
			// append the input to the inputs
			inputs = append(inputs, input)
		}
	}

	// create a new output for the receiver
	output := TXoutput{amount, nil}
	output.LockAddress(toAddr) // 使用收款人的地址锁定交易输出
	// append the output to the outputs
	outputs = append(outputs, output)

	// if the actualBalance is greater than the amount,
	// we need to send the change back to the sender
	if actualBalance > amount {
		// create a new output for the sender
		output := TXoutput{actualBalance - amount, nil}
		output.LockAddress(fromAddr) // 使用付款人的地址锁定交易输出
		// append the output to the outputs
		outputs = append(outputs, output)
	}

	// create a new transaction
	tx := Transaction{nil, inputs, outputs}
	tx.ID = tx.Hash()

	blockchain.SignTransaction(&tx, senderKeyPair.PrivateKey)
	return &tx
}

// Verify verifies the transaction input
func (tx *Transaction) Verify(inputTxs map[string]*Transaction) bool {
	if tx.IsCoinbase() {
		// coinbase tx has no input, so it's valid
		return true
	}

	// 这里不是很理解为什么还要判断一次是否在mapping
	for _, input := range tx.In {
		// 如果交易输入的TXid不在mapping中，说明交易输入的TXid是无效的
		if inputTxs[string(input.TXid)] == nil {
			fmt.Println("ERROR: Previous transaction is not correct")
			panic("ERROR: Previous transaction is not correct")
		}
	}

	txcopy := tx.trimmedCopy()

	curve := secp256k1.S256() // secp256k1 椭圆曲线

	// 参考构建交易的时候的签名过程，重新构建交易输入的签名来验证两者是否一致
	// 参考这个方法的实现 (tx *Transaction) sign
	for inputIdx, input := range tx.In {
		prevtx := inputTxs[string(input.TXid)] // 获取交易输入引用的上一笔交易的结构体
		// txcopy.In[inputIdx].Signature = nil		// 将交易输入的签名置空, 但是并不需要，因为在trimmedCopy中已经置空了
		txcopy.In[inputIdx].Pubkey = prevtx.Out[input.Voutindex].PublickeyHash // 将交易输入的公钥置为上一笔交易输出的公钥哈希
		txcopy.ID = txcopy.Hash()                                              // 重新计算交易的哈希值

		r, s := big.Int{}, big.Int{} // 用来存储交易输入的签名
		signatureLength := len(input.Signature)
		r.SetBytes(input.Signature[:(signatureLength / 2)])
		s.SetBytes(input.Signature[(signatureLength / 2):])

		// get the public key from the signature
		ecPubkeyX := big.Int{}
		ecPubkeyY := big.Int{}

		// 从签名中获取公钥
		// XXX: 这里会有问题，因为input中的pubkey实际上是pubkeyhash，而不是pubkey, 可能会导致验证失败
		pubkeyLength := len(input.Pubkey)
		ecPubkeyX.SetBytes(input.Pubkey[:(pubkeyLength / 2)])
		ecPubkeyY.SetBytes(input.Pubkey[(pubkeyLength / 2):])

		rawPubkey := ecdsa.PublicKey{Curve: curve, X: &ecPubkeyX, Y: &ecPubkeyY}

		// 只要有一个交易输入的签名验证失败，就返回false
		if !ecdsa.Verify(&rawPubkey, txcopy.ID, &r, &s) {
			return false
		}

		txcopy.In[inputIdx].Pubkey = nil // 将交易输入的公钥置空 用于下一次循环
	}

	return true
}
