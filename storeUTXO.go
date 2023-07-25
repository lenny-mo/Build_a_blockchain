package main

import "github.com/boltdb/bolt"

const (
	// 每个比特币交易都会消费一些UTXO，并生成一些新的UTXO。
	// 所有的UTXO集合实际上代表了比特币网络的当前状态
	UTXOBucket = "chainstate"
)

// 存储UTXO
type UTXOSet struct {
	Blockchain *Blockchain
}

func (u *UTXOSet) StoreUTXO() {

	utxodb := u.Blockchain.db

	utxo_bucketName := []byte(UTXOBucket)

	err := UTXOBucket.Update(func(tx *bolt.Tx) error {
		
	})
	
}



