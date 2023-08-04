package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/boltdb/bolt"
)

const (
	DBFILE      = "blockchain.db" // 数据库文件名
	BLOCKBUCKET = "blocks"        // 区块桶名
)

type Blockchain struct {
	topHash []byte   // 最新区块的哈希值
	db      *bolt.DB // 数据库
}

type BlockchainIterator struct {
	currentHash []byte   // 当前区块的哈希值
	db          *bolt.DB // 数据库
}

// ------------------------- Blockchain Basic -------------------------

// GetBlockchain returns the latest block hash
//
// 返回最新区块的哈希值
func (bc *Blockchain) GetTopHash() []byte {
	return bc.topHash
}

// CreateBlockchain creates a new blockchain DB
//
// 创建一个新的区块链并且添加一个创世区块
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

		// if bucket is nil, blockchain doesnt exist, we then create a new blockchain
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

	UTXOset := UTXOSet{&blockchain} // 创建UTXO集合
	UTXOset.StoreUTXO()             // 存储UTXO

	return &blockchain
}

// AddBlock update the latest block into the blockchain
//
// 根据最新区块的哈希值和交易列表，创建一个新的区块，并更新区块链
func (bc *Blockchain) AddBlock(txs []*Transaction) (bool, *Block) {
	var tophash []byte
	var latestHeight int64

	// 验证交易序列中的所有交易都是有效的
	for _, tx := range txs {
		if !bc.VerifyTransaction(tx) {
			fmt.Printf("This Transaction is invalid: %v\n", tx)
			panic(errors.New("this transaction is invalid"))
		} else {
			fmt.Printf("This Transaction is valid\n: %v\n", tx)
		}
	}

	// get the latest block hash
	err := bc.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BLOCKBUCKET))
		tophash = bucket.Get([]byte("latest")) // 获取最新区块的哈希值

		blockdata := bucket.Get(tophash)
		block := Deserialize(blockdata)
		latestHeight = block.Height // 获取最新区块的高度
		return nil
	})
	if err != nil {
		panic(err)
	}

	// create a new block according to the latest block hash and transactions
	newBlock := NewBlock(tophash, txs, latestHeight+1)

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

	return true, newBlock
}

// ---------------------------- 以下是区块链迭代器 ----------------------------

// Iterator returns a BlockchainIterator
//
// 创建一个区块链迭代器
func (bc *Blockchain) Iterator() *BlockchainIterator {
	return &BlockchainIterator{bc.topHash, bc.db}
}

// Next returns the next block of the blockchain according to the current hash
//
// 返回当前区块链迭代器所指向的区块，并且把迭代器的currentHash指向下一个区块
func (bit *BlockchainIterator) Next() *Block {
	var block *Block

	// get the block from the database
	// view method does not allow to modify the database
	err := bit.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BLOCKBUCKET))
		// get a block according to the current hash
		serializedBlock := bucket.Get(bit.currentHash)
		block = Deserialize(serializedBlock)
		return nil
	})
	if err != nil {
		panic(err)
	}

	// update the current hash
	bit.currentHash = block.PrevBlockHash

	return block
}

// IterateBlockchain iterates the blockchain
//
// 迭代区块链并且打印
func (bc *Blockchain) IterateBlockchain() {
	iterator := bc.Iterator()
	for {
		block := iterator.Next()
		fmt.Printf("Block number: %v\n", block.String())

		// when the previous block hash is empty, then the genesis block is reached
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

}

// FindUnspendTransaction finds all unspend transactions according to the address
//
// 根据给定的地址，找到这个地址所没有花费的输出所在的交易
func (bc *Blockchain) FindUnspendTransaction(pubkeyHash []byte) []*Transaction {
	// 关于addr的所有未花费的交易，在这些交易中一定包含有某个output是属于addr的
	// 但是，这些交易中可能还有其他output也是属于addr的，所以我们需要遍历这些交易，找到所有属于addr的output
	unsepentTXs := []*Transaction{}

	// 存储一笔交易中所有被使用的输出; map[交易ID][]int, []int对应的是交易中的输出索引
	spendTxos := make(map[string][]int)

	bcIterator := bc.Iterator()

	for {
		// iterate over all blocks from the newest to the oldest
		block := bcIterator.Next()

		// iterate over all transactions in one block
		for _, tx := range block.Transactions {
			txID := string(tx.ID)

			// iterate over all outputs in one transaction
		Outputs:
			for outIdx, output := range tx.Out {
				// if spendTxos[txID] != nil, it means that some outputs in this transaction have been used
				// we need to put the used output index into the slice
				if spendTxos[txID] != nil {
					// iterate over all used outputs in spendTxos[txID] to check whether the output has been used
					for _, spentOutput := range spendTxos[txID] {
						// it means that the outIdx has been used
						if spentOutput == outIdx {
							continue Outputs // outIdx has been used, so we skip to the next outIdx
						}
					}
				}

				// if the outout not been used,
				// if the output can be unlocked by the address,
				// it means that the address has not spent this output
				if output.CanBeUnlockedWith(pubkeyHash) {
					// eg. tx #3 有3笔输出，其中第一笔输出被使用了，那么spendTxos[tx #3] = []int{0}
					// 剩下的两笔输出中只有第二笔是给bob的，所以unsepentTXs = []*Transaction{tx #3}
					// 说明tx #3中存在关于bob的未花费输出
					unsepentTXs = append(unsepentTXs, tx)
				}
			}

			// tx can have input only if it is not a coinbase transaction
			if !tx.IsCoinbase() {
				for _, input := range tx.In {
					// if the input can unlock the output with the address,
					// it means that the address has spent the output
					if input.CanUnlockOutputWith(pubkeyHash) {
						inputTxID := string(input.TXid)
						// inputTxID 记录了上一笔交易的ID
						// input.Voutindex 记录了上一笔交易中的具体哪一笔输出被使用了
						spendTxos[inputTxID] = append(spendTxos[inputTxID], input.Voutindex) // 记录这个交易中被使用的输出
					}
				}
			}
		}

		// 如果到了创世区块，停止遍历, 创世区块的PrevBlockHash是空的
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return unsepentTXs
}

// FindUTXO finds all unspent transaction outputs according to the address
//
// 根据给定的地址，找到这个地址在当前区块链中所没有花费的输出，需要使用FindUnspendTransaction函数
func (bc *Blockchain) FindUTXO(pubkeyHash []byte) []*TXoutput {
	UTXOs := []*TXoutput{}

	unspentTxs := bc.FindUnspendTransaction(pubkeyHash)

	// iterate over all transactions
	for _, tx := range unspentTxs {
		// iterate over all outputs in one transaction
		for _, output := range tx.Out {
			// if the output can be unlocked by the address,
			// it means that this output belongs to the address
			if output.CanBeUnlockedWith(pubkeyHash) {
				UTXOs = append(UTXOs, &output)
			}
		}
	}

	return UTXOs
}

// FindSpendableOutputs finds all unspent transaction outputs according to the address and the amount
//
// 根据给定的地址和金额，找到这个地址在当前区块链中所没有花费的输出，
// 根据给定的金额，找到这个地址所没有花费的输出中，能够满足给定金额的输出
func (bc *Blockchain) FindSpendableOutputs(pubkeyHash []byte, amount int) (int, map[string][]int) {
	// map[交易ID][]int, []int对应的是交易中的输出索引,
	// 也就是说，需要记录addr 未花费的输出所在的交易ID和输出索引
	unspentOutputs := make(map[string][]int)

	unspentTxs := bc.FindUnspendTransaction(pubkeyHash) // 获取addr的所有未花费的交易

	sum := 0

TxLoop:
	// iterate over all transactions
	for _, tx := range unspentTxs {
		txID := string(tx.ID)

		// iterate over all outputs in one transaction
		for outputIdx, output := range tx.Out {
			// if the output can be unlocked by the address,
			// it means that this output belongs to the address
			// meanwhile, the sum of the outputs is less than the amount
			// else it should be skipped
			if output.CanBeUnlockedWith(pubkeyHash) && sum < amount {
				sum += output.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outputIdx)

				if sum >= amount {
					break TxLoop // 获取到了足够的金额，停止遍历Tx
				}
			}
		}
	}

	// 有可能，addr的所有未花费的交易中，没有足够的金额
	// 此时，sum < amount

	return sum, unspentOutputs
}

// SignTransaction signs a transaction
//
// 对交易进行签名
func (bc *Blockchain) SignTransaction(tx *Transaction, privatekey ecdsa.PrivateKey) {

	txmapping := make(map[string]*Transaction) // 记录tx的所有输入所在的交易

	// iterate all inputs in one transaction
	// 找到tx的所有输入所在的交易
	for _, input := range tx.In {
		// get the transaction which the input belongs to
		inputTx, err := bc.FindTxByID(input.TXid)
		if err != nil {
			fmt.Printf("Transaction is not found: %v\n", err)
			panic(err)
		}

		// add the transaction into the mapping
		txmapping[string(input.TXid)] = inputTx
	}

	tx.sign(privatekey, txmapping)
}

// sign signs a transaction using a private key
//
// 对交易进行签名
func (tx *Transaction) sign(privatekey ecdsa.PrivateKey, mapping map[string]*Transaction) {
	if tx.IsCoinbase() {
		return
	}

	// 检查tx的所有输入所在的交易是否都存在
	for _, input := range tx.In {
		if mapping[string(input.TXid)] == nil {
			fmt.Printf("Transaction is not found: %v\n", errors.New("Transaction is not found"))
			panic(errors.New("Transaction is not found"))
		}
	}

	// 对当前的tx进行复制, 其中Inputs的Signature和Pubkey设置为nil
	txcopy := tx.trimmedCopy()

	// 对txcopy的所有输入进行签名
	for inIdx, input := range txcopy.In {
		preTx := mapping[string(input.TXid)]
		txcopy.In[inIdx].Signature = nil
		txcopy.In[inIdx].Pubkey = preTx.Out[input.Voutindex].PublickeyHash // 设置Pubkey为上一笔交易的输出的公钥哈希

		txcopy.ID = txcopy.Hash()                                    // 重新计算交易的哈希值
		r, s, err := ecdsa.Sign(rand.Reader, &privatekey, txcopy.ID) // 对交易的哈希值进行签名
		if err != nil {
			panic(err)
		}

		signature := append(r.Bytes(), s.Bytes()...)
		tx.In[inIdx].Signature = signature
	}
}

// trimmedCopy returns a copy of the transaction with all inputs' signature and pubkey set to nil
//
// 把交易的所有输入的Signature和Pubkey设置为nil，返回一个交易的副本
func (tx *Transaction) trimmedCopy() Transaction {
	var inputs []TXinput
	var outputs []TXoutput

	for _, input := range tx.In {
		inputs = append(inputs, TXinput{input.TXid, input.Voutindex, nil, nil})
	}

	for _, output := range tx.Out {
		outputs = append(outputs, TXoutput{output.Value, output.PublickeyHash})
	}

	txCopy := Transaction{tx.ID, inputs, outputs}

	return txCopy
}

func (bc *Blockchain) FindTxByID(txID []byte) (*Transaction, error) {
	// iterate over all blocks from the newest to the oldest
	bcIterator := bc.Iterator()

	for {
		block := bcIterator.Next()

		// iterate over all transactions in one block
		for _, tx := range block.Transactions {
			// Equal 方法比Compare速度更快
			if bytes.Equal(tx.ID, txID) {
				return tx, nil
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return &Transaction{}, errors.New("Transaction is not found")
}

// VerifyTransaction verifies the transaction using the public key
func (bc *Blockchain) VerifyTransaction(tx *Transaction) bool {
	prevTxs := make(map[string]*Transaction) // 记录tx的所有输入所在的交易

	// 如果是coinbase交易，不需要验证
	if tx.IsCoinbase() {
		return true
	}

	for _, input := range tx.In {
		prevTx, err := bc.FindTxByID(input.TXid) // 找到tx的输入所在的交易
		if err != nil {
			fmt.Printf("Transaction is not found: %v\n", err)
			panic(err)
		}

		prevTxs[string(input.TXid)] = prevTx // 把交易放入map中
	}

	return tx.Verify(prevTxs) // 验证所有的输入是否满足条件
}

// FindAllUTXO finds all unspent transaction outputs
//
// 找到所有未花费的输出; map [交易ID] 所有未花费的输出
func (bc *Blockchain) FindAllUTXO() map[string]TXoutputSlice {
	// nil值slice可以使用append函数
	UTXO := make(map[string]TXoutputSlice) // map [交易ID] 所有未花费的输出

	spentTxs := make(map[string][]int) // 记录一笔交易中所有被使用的输出

	bcIterator := bc.Iterator()

	for {
		block := bcIterator.Next() // get current block

		// iterate over all transactions in one block
		for _, tx := range block.Transactions {
			txID := string(tx.ID) // 获取交易ID

			// 如果一个交易中的某个输出被使用了，跳过这个交易遍历下一个输出
		Outputs:
			// 遍历交易中的所有输出, 判断是否被使用
			for outputIdx, output := range tx.Out {

				// 如果当前的交易ID存在于spentTxs中，说明这笔交易中的某个输出已经被使用过了, 需要
				if outputOfSpentTx, ok := spentTxs[txID]; ok {
					// 则遍历这笔交易中所有被使用的输出的索引
					for _, outputIdxOfSpentTx := range outputOfSpentTx {
						// 判断当前交易被使用的输出的索引，是否等于当前交易的outputIdx
						// 检查当前正在检查的交易输出的索引（outputIdx）是否等于已经被使用过的某个输出的索引（outputIDOfSpentTx）。
						// 如果等于，说明当前正在检查的交易输出已经被使用过了，因此它不能被视为 UTXO。
						if outputIdxOfSpentTx == outputIdx {
							// 说明当前的outputIdx是被使用的, 跳过当前的outputIdx, 继续遍历下一个outputIdx
							continue Outputs
						}
					}
				}

				// update the UTXO[txID] slice
				outputslice := UTXO[txID]
				outputslice = append(outputslice, output)
				UTXO[txID] = outputslice
			}

			// 遍历交易当中的所有输入
			if !tx.IsCoinbase() {
				for _, input := range tx.In {
					prevTxID := string(input.TXid)                                   // 获取input所使用的上一个交易的ID
					spentTxs[prevTxID] = append(spentTxs[prevTxID], input.Voutindex) // 记录这个交易中被使用的输出
				}
			}
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return UTXO
}

// GetLatestHeight returns the latest block height
//
// 获取最新的区块高度 , 创世纪区块的高度为0
func (blockchain *Blockchain) GetLatestHeight() (int64, error) {
	var block *Block

	err := blockchain.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKBUCKET)) // blockBucket name
		lastHash := b.Get([]byte("latest"))
		blockData := b.Get(lastHash)
		block = Deserialize(blockData)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return block.Height, nil
}
