package main

import "github.com/boltdb/bolt"

const (
	DBFILE      = "blockchain.db"
	BLOCKBUCKET = "blocks"
)

type Blockchain struct {
	topHash []byte   // 最新区块的哈希值
	db      *bolt.DB // 数据库
}

// CreateBlockchain creates a new blockchain DB
//
// 创建一个新的区块链数据库
func CreateBlockchain() *Blockchain {
	// 0600 文件拥有者具有读写权限，其他人无任何权限
	boltDB, err := bolt.Open(DBFILE, 0600, nil)
	if err != nil {
		panic(err)
	}

	var tophash []byte // 最新区块的哈希值
	// update the blockchain
	err = boltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BLOCKBUCKET))

		// if bucket is nil, then create a new blockchain
		if bucket == nil {
			// create a genesisblock
			genesisBlock := GenesisBlock()

			// 创建一个新的bucket
			bucket, err = tx.CreateBucket([]byte(BLOCKBUCKET))
			if err != nil {
				panic(err)
			}

			// put the genesis block hash and genesis block into the bucket
			err = bucket.Put(genesisBlock.Hash, genesisBlock.Serialize())
			if err != nil {
				panic(err)
			}
			// put the genesis block hash and latest into the bucket
			err = bucket.Put([]byte("latest"), genesisBlock.Hash)
			if err != nil {
				panic(err)
			}
			tophash = genesisBlock.Hash

		} else {
			// genesis block already exists,
			// get the latest block hash
			tophash = bucket.Get([]byte("latest"))
		}
		return nil
	})

	if err != nil {
		panic(err)
	}

	blockchain := Blockchain{topHash: tophash, db: boltDB}

	return &blockchain
}

// AddBlock update the latest block into the blockchain
//
// 根据最新区块的哈希值和交易列表，创建一个新的区块，并更新区块链
func (bc *Blockchain) AddBlock(txs []*Transaction) bool {
	var tophash []byte

	// get the latest block hash
	err := bc.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BLOCKBUCKET))
		tophash = bucket.Get([]byte("latest"))
		return nil
	})
	if err != nil {
		panic(err)
	}

	// create a new block according to the latest block hash and transactions
	newBlock := NewBlock(tophash, txs)

	// update the blockchain
	bc.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BLOCKBUCKET))
		// put the new block and block hash into the bucket
		err := bucket.Put(newBlock.Hash, newBlock.Serialize())
		if err != nil {
			panic(err)
		}

		// update the latest block hash
		err = bucket.Put([]byte("latest"), newBlock.Hash)
		if err != nil {
			panic(err)
		}

		// update the latest block hash
		bc.topHash = newBlock.Hash

		return nil
	})

	if err != nil {
		panic(err)
	}

	return true
}
