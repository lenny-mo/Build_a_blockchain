package main

import (
	"fmt"

	"github.com/boltdb/bolt"
)

const (
	// 每个比特币交易都会消费一些UTXO，并生成一些新的UTXO。
	// 所有的UTXO集合实际上代表了比特币网络的当前状态
	UTXOBUCKET = "chainstate"
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

	utxo_bucketName := []byte(UTXOBUCKET)

	// 1. 创建bucket
	err := utxodb.Update(func(tx *bolt.Tx) error {
		// 删除bucket
		err := tx.DeleteBucket(utxo_bucketName)
		if err != nil && err != bolt.ErrBucketNotFound {
			return fmt.Errorf("delete bucket %s failed, %w", UTXOBUCKET, err)
		}

		// 创建bucket
		_, err = tx.CreateBucket(utxo_bucketName)
		if err != nil {
			return fmt.Errorf("create bucket %s failed, %w", UTXOBUCKET, err)
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

// FindUTXOByPubkeyHash finds all UTXO for a public key hash
//
// 根据公钥哈希查找UTXO, 使用之前必须通过StoreUTXO函数 创建bucket，否则会报错
func (uset *UTXOSet) FindUTXOByPubkeyHash(pubkeyHash []byte) TXoutputSlice {

	var utxos TXoutputSlice // 用于存储查找到的UTXO

	db := uset.Blockchain.db // 小写开头，包内可见

	// 实现
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UTXOBUCKET)) // 获取bucket, 使用之前必须通过StoreUTXO函数 创建bucket

		dbCursor := b.Cursor() // 创建游标用于遍历bucket

		// k: 交易ID, v: 交易输出切片
		for k, v := dbCursor.First(); k != nil; k, v = dbCursor.Next() {

			outputslice := DeserializeOutputSlice(v)

			// 遍历交易输出切片, 查找公钥哈希可以解锁的UTXO
			for _, output := range outputslice {
				if output.CanBeUnlockedWith(pubkeyHash) {
					utxos = append(utxos, output)
				}
			}
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	return utxos
}

// UpdateUTX sets the UTXO set with the UTXO in the new block
//
// 把新添加的区块中的UTXO添加到UTXO集合中, 并且更新当前的区块
func (u *UTXOSet) UpdateUTXO(block *Block) {
	db := u.Blockchain.db

	// 更新数据库中的UTXO
	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(UTXOBUCKET))

		// 遍历新增区块中的所有交易
		for _, tx := range block.Transactions {
			// 非coinbase交易, 遍历交易输入
			if !tx.IsCoinbase() {
				// 遍历新增区块中的当前交易的输入
				for _, input := range tx.In {
					tempOutputSlice := TXoutputSlice{} // 存储当前交易输入引用的上一笔交易中，没有使用的输出

					txID := input.TXid           // 当前交易输入引用的交易ID
					outputTxBytes := b.Get(txID) // 根据txID获取当前交易的交易输出切片的字节数组

					outputSlice := DeserializeOutputSlice(outputTxBytes)

					// 遍历上一笔交易的每一个输出，如果当前交易的输入引用了上一笔交易的输出，那么就删除这个输出
					for outputIdx, output := range outputSlice {
						// 如果当前交易没有使用这个输出，把这个输出暂存
						if outputIdx != input.Voutindex {
							tempOutputSlice = append(tempOutputSlice, output)
						}
					}

					// 如果上一笔交易的所有输出都被使用了，那么就删除这笔交易
					if len(tempOutputSlice) == 0 {
						err := b.Delete(input.TXid)
						if err != nil {
							panic(fmt.Errorf("delete transaction fail, %w", err))
						}
					} else {
						// 如果上一笔交易的所有输出没有被使用完，那么就更新这笔交易
						err := b.Put(input.TXid, tempOutputSlice.Serialize())
						if err != nil {
							panic(fmt.Errorf("update transaction fail, %w", err))
						}
					}
				}

				// 遍历交易输出
				newOutputSlice := TXoutputSlice{}

				// 把新区块当前交易的所有输出添加到newOutputSlice中
				for _, output := range tx.Out {
					newOutputSlice = append(newOutputSlice, output)
				}

				err := b.Put(tx.ID, newOutputSlice.Serialize())
				if err != nil {
					panic(fmt.Errorf("update transaction fail, %w", err))
				}
			}
		}
		return nil
	})

	if err != nil {
		panic(fmt.Errorf("update utxo fail, %w", err))
	}
}
