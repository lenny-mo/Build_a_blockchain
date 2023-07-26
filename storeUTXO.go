package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

const (
	// 每个比特币交易都会消费一些UTXO，并生成一些新的UTXO。
	// 所有的UTXO集合实际上代表了比特币网络的当前状态
	UTXOBucket = "chainstate"
)

// 存储UTXO
type UTXOSet struct {
	Blockchain *Blockchain
}

// StoreUTXO deletes the old UTXO set and stores the current one into chainstate bucket
//
// 删除并且创建UTXO bucket，然后将UTXO存储到bucket中
func (u *UTXOSet) StoreUTXO() error {

	utxodb := u.Blockchain.db

	utxo_bucketName := []byte(UTXOBucket)

	// 1. 创建bucket
	err := utxodb.Update(func(tx *bolt.Tx) error {
		// 删除bucket
		err := tx.DeleteBucket(utxo_bucketName)
		if err != nil {
			return fmt.Errorf("delete bucket %s failed, %w", UTXOBucket, err)
		}

		// 创建bucket
		_, err = tx.CreateBucket(utxo_bucketName)
		if err != nil {
			return fmt.Errorf("create bucket %s failed, %w", UTXOBucket, err)
		}

		return nil

	})

	if err != nil {
		return fmt.Errorf("update utxo bucket failed, %w", err)
	}

	// 2. 从区块链中获取所有的UTXO
	UTXO := u.Blockchain.FindAllUTXO()

	// 3. 将UTXO存储到bucket中
	err = utxodb.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(utxo_bucketName) // 获取bucket

		// 遍历UTXO mapping, 获取每个交易的UTXO
		for txID, utxos := range UTXO {
			key := []byte(txID)

			err := b.Put(key, utxos.Serialize()) // store utxo in bucket
			if err != nil {
				return fmt.Errorf("put utxo failed, %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("update utxo bucket failed, %w", err)
	}

	return nil
}
