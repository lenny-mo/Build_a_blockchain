package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"

	"github.com/boltdb/bolt"
)

type Version struct {
	Version      int    // 版本号
	LatestHeight int64  // 最新区块的高度
	Addrfrom     string // 发送方地址
}

const (
	// 常量只能是布尔型、数字型（整数型、浮点型和复数型）和字符串型
	// 切片、函数、指针、接口、结构体等都不可以是常量
	NODEVERSION   = 1
	COMMANDLENGTH = 16 // 命令的长度
)

var (
	// 公链的种子节点会预先设置好一些种子节点的地址，也可以向外部获取种子节点的地址
	// 通过udp广播的方式，让其他节点知道自己的存在``
	KnownNodes     = []string{"localhost:3000"} // 种子节点列表
	CurrentNode    = ""                         // 当前节点
	BlockInTransit [][]byte                     // 传输中的区块
)

func (ver *Version) String() string {
	str := fmt.Sprintf("Version: %d\n", ver.Version)
	str += fmt.Sprintf("LatestHeight: %d\n", ver.LatestHeight)
	str += fmt.Sprintf("Addrfrom: %s\n", ver.Addrfrom)
	return str
}

// StartServer starts a node
//
// 给定一个端口号，持续监听这个端口号
func StartServer(nodeID, minderAddr string, blockchain *Blockchain) bool {
	nodeAddr := fmt.Sprintf("localhost:%s", nodeID)
	CurrentNode = nodeAddr

	// 一个程序监听一个地址（比如"localhost:3000"）只是指这个程序已经准备好接收和处理发往这个地址的网络请求
	listener, err := net.Listen("tcp", nodeAddr) // 监听当前节点的地址
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	fmt.Printf("start to listen this address: %s\n", nodeAddr)

	// 如果当前节点不是种子节点, 需要向种子节点发送版本信息, 让种子节点知道新节点的存在
	// 这里 nodeAddr 就是localhost:3000
	if nodeAddr != KnownNodes[0] {
		fmt.Println("current node is not a seed node, need to send version to seed node")
		sendVersion(KnownNodes[0], blockchain)
	}

	for {
		connect, err := listener.Accept() // 接收到一个连接
		if err != nil {
			panic(err)
		}
		// 非种子节点向种子节点发送版本信息后，
		// 种子节点需要处理这个新节点的连接请求
		go handleConnection(connect, blockchain)
	}

}

// sendVersion sends version message to a node
//
// 向一个节点发送版本信息
func sendVersion(toAddr string, bc *Blockchain) bool {
	latestHeight, _ := bc.GetLatestHeight()

	version := Version{
		NODEVERSION,
		latestHeight,
		CurrentNode, // 当前节点的地址
	}

	payload := EncodeEverything(version)                     // convert into bytes
	request := append(commandToBytes("version"), payload...) // add command to  the front
	return sendData(toAddr, request)                         //向对方发送数据
}

// EncodeEverything encodes everything
//
// 传入任意类型的数据，返回编码后的字节数组
func EncodeEverything(data interface{}) []byte {
	var buffer bytes.Buffer

	encoder := gob.NewEncoder(&buffer)
	err := encoder.Encode(data)
	if err != nil {
		panic(err)
	}

	return buffer.Bytes()
}

// commandToBytes converts a command to bytes
//
// 把一个命令转换成长度固定的字节数组；考虑是否可以使用hash, 把命令转换成固定长度的哈希值
func commandToBytes(cmd string) []byte {
	var bytes [COMMANDLENGTH]byte

	// 把命令转换成字节数组, 剩余没有填充的部分默认为0
	for i, char := range cmd {
		bytes[i] = byte(char)
	}

	return bytes[:]
}

// bytesToCommand converts bytes to command
//
// 把字节数组转换成命令
func bytesToCommand(bytes []byte) string {
	var command = make([]byte, 0)

	for _, v := range bytes {
		if v != 0x0 {
			command = append(command, v)
		} else {
			break
		}
	}

	return string(command[:])
}

// sendData sends data to a node
//
// 向一个种子节点发送数据, 如果节点不存在, 那么就把这个节点从种子节点列表中删除
func sendData(toAddr string, data []byte) bool {
	connect, err := net.Dial("tcp", toAddr) // 使用TCP协议连接到toAddr

	// 如果这个地址无法连接，那么就把这个地址从种子节点列表中删除
	if err != nil {
		fmt.Printf("address %s is not available\n", toAddr)

		updateNodes := []string{}

		// 遍历种子节点列表，把无法连接的节点剔除
		// updateNodes列表中就包含了所有可以连接的节点，而无法连接的节点（即toAddr）被排除在外。
		for _, node := range KnownNodes {
			if node != toAddr {
				updateNodes = append(updateNodes, node)
			}
		}

		KnownNodes = updateNodes // 返回剔除之后的种子节点列表
		return false
	}
	defer connect.Close()

	// 如果连接建立成功，向对方发送数据
	_, err = io.Copy(connect, bytes.NewReader(data)) // send data to connect
	if err != nil {
		panic(err)
	}

	return true
}

// handleConnection handles connection
//
// 从连接中读取数据，然后根据命令执行对应的函数
func handleConnection(conn net.Conn, bc *Blockchain) {

	// 1. 从连接中读取数据
	request, err := io.ReadAll(conn)
	if err != nil {
		fmt.Println("cannot connect to addr, %w", err)
		panic(err)
	}

	// 2. 从request中解析出命令
	command := bytesToCommand(request[:COMMANDLENGTH])

	// 3. 接收到来自其他节点的命令，根据命令执行对应的函数
	switch command {
	case "version":
		// 其他节点向当前节点发送version信息，用于比较当前节点和其他节点的区块链高度
		fmt.Println("receive version message")
		handleVersion(request, bc)
	case "getblocks":
		// 其他节点发送的getblocks信息，想要获取当前节点的区块
		fmt.Println("receive getblocks message")
		handleGetBlocks(request, bc)
	case "inv":
		// 其他节点发送的inv信息，包含了对方节点的区块链中的所有区块的hash值
		fmt.Println("receive inv message")
		handleInv(request, bc)
	case "getdata":
		// 其他节点发送的getdata请求，想要获取当前节点的区块链中的某个区块
		fmt.Println("receive getdata message")
		handleGetData(request, bc)
	case "block":
		// 其他节点发送的block信息，包含了对方节点的区块链中的某个区块，当前节点需要把这个区块添加到自己的区块链中
		fmt.Println("receive block message")
		handleBlock(request, bc)
	}
}

// handleVersion handles version message
//
// 处理version信息
func handleVersion(request []byte, bc *Blockchain) {
	// 1. 解码version信息
	var buff bytes.Buffer
	var payload Version // payload 指代在一个数据包或消息中，实际携带的、对于最终用户有意义的数据

	decoder := gob.NewDecoder(&buff)
	buff.Write(request[COMMANDLENGTH:])
	// 因为Decode()方法需要在已有的内存空间（也就是你传入的那个变量）上直接进行修改，
	// 而不是创建一个新的变量。如果你传入一个变量（而不是指针），
	// Decode()方法会在一个新的内存空间上进行操作，这个新的内存空间只在Decode()方法内部存在，
	// 当方法返回后，这个新的内存空间就会被释放，你在方法外部是无法访问到这个新的内存空间的
	err := decoder.Decode(&payload) // 解压version信息到payload中
	if err != nil {
		panic(err)
	}

	localHeight, _ := bc.GetLatestHeight() // 获取当前节点的区块高度，也就是种子节点的区块高度
	remoteHeight := payload.LatestHeight   // 获取发送方的区块高度

	fmt.Println("种子节点当前高度: ", localHeight)
	fmt.Println("remoteHeight: ", remoteHeight)
	fmt.Println("remoteVersion struct info: ", payload.String())

	// 2. 根据区块高度判断当前节点和发送方的区块数据是否同步
	if remoteHeight > localHeight {
		// 如果对方的区块高度大于当前节点的区块高度，那么就向对方请求区块数据
		// TODO: getBlocksFrom()
		getBlocksFrom(payload.Addrfrom)
	} else {
		// 如果对方的区块高度小于当前节点的区块高度，那么就向对方发送区块数据
		sendVersion(payload.Addrfrom, bc)
	}

	// 如果发送方的地址不在种子节点列表中，那么就把发送方的地址添加到种子节点列表中
	if !isKnownNode(payload.Addrfrom) {
		KnownNodes = append(KnownNodes, payload.Addrfrom)
	}

	fmt.Println("Current Known Nodes: ", KnownNodes)
	fmt.Println("handleVersion func complete")
}

// isKnownNode checks if a node is known
//
// 判断一个节点是否在种子节点列表中
func isKnownNode(addr string) bool {
	for _, node := range KnownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

type GetBlocks struct {
	AddrFrom string // 请求方的地址
}

// getBlocksFrom gets blocks from a node
//
// 从addr 地址获取缺少的区块数据，latestHeight是发送方的区块高度
func getBlocksFrom(toAddr string) {
	// 1. 构建getblocks命令
	// payload 内部包含了当前节点的地址
	// XXX: 按照视频说的，这里的CurrentNode似乎有问题？
	payload := EncodeEverything(GetBlocks{AddrFrom: CurrentNode})
	request := append(commandToBytes("getblocks"), payload...)

	// 2. 向对方addr发送getblocks命令
	sendData(toAddr, request)
}

// handleGetBlocks handles getblocks message from other nodes
//
// 处理其他节点发送过来的getblocks命令, 把当前节点的所有区块hash发送给请求方
func handleGetBlocks(request []byte, blockchain *Blockchain) {
	// 1. 解码getblocks命令
	var buff bytes.Buffer
	var payload GetBlocks // 包含了请求方的地址

	decoder := gob.NewDecoder(&buff)    // 构建解码器
	buff.Write(request[COMMANDLENGTH:]) // 将request中的数据写入到buff中
	err := decoder.Decode(&payload)     // 解码buff中的数据到payload中
	if err != nil {
		panic(err)
	}

	// 2. 获取当前节点的所有区块hash？？？
	// 为什么要获取所有区块的hash，而不是只发送对方缺少的区块hash
	blockHashes := blockchain.GetBlockHashes()

	// 3. 向请求方发送inv命令，包含了当前节点的所有区块hash
	sendInv(payload.AddrFrom, "block", blockHashes) // 把当前节点的所有区块hash发送给请求方

	fmt.Println("handleGetBlocks func complete")
}

// GetBlockHashes returns all block hashes
// TODO: move to blockchain.go
// 获取当前节点区块链的所有区块的hash
func (bc *Blockchain) GetBlockHashes() [][]byte {
	var blocks [][]byte
	iter := bc.Iterator()

	for {
		block := iter.Next()
		blocks = append(blocks, block.Hash)
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}

	return blocks
}

type INV struct {
	AddrFrom string
	Type     string
	Items    [][]byte // 包含了所有区块的hash
}

// sendInv sends inv message to a node
//
// 把当前节点的所有区块hash发送给请求方
func sendInv(toAddr, kind string, items [][]byte) {
	// 1. 构建inv命令
	payload := EncodeEverything(INV{AddrFrom: CurrentNode, Type: kind, Items: items})
	request := append(commandToBytes("inv"), payload...)

	// 2. 向addr发送inv命令
	sendData(toAddr, request)
}

// handleInv handles inv message from other nodes
//
// 接收到对方节点的所有区块hash, 当前节点接收到inv命令后，把缺少的区块hash发送给对方节点
func handleInv(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var paypload INV

	decoder := gob.NewDecoder(&buff)
	buff.Write(request[COMMANDLENGTH:])
	err := decoder.Decode(&paypload)
	if err != nil {
		panic(err)
	}

	// 打印接收到的inv信息
	fmt.Println("Received inventory with ", len(paypload.Items), " from ", paypload.AddrFrom)

	// 处理所有区块的hash
	if paypload.Type == "block" {
		BlockInTransit = paypload.Items      // 将所有区块的hash存储到BlockInTransit中
		latestBlockHash := paypload.Items[0] // 获取最新的区块hash

		// XXX: 向对方节点请求最新的区块数据
		getBlockData(paypload.AddrFrom, "block", latestBlockHash) // 向种子节点发送getdata命令，请求最新的区块数据

		newInTransit := [][]byte{}

		// XXX:??? 不理解
		// 逐个请求区块数据，每请求一个数据，就把对应的区块哈希从列表中移除。
		// 这样，BlockInTransit 变量就始终保持了需要请求的区块哈希列表，防止重复请求已经得到的数据。
		for _, b := range BlockInTransit {
			// 剔除掉已经请求过的区块hash
			if !bytes.Equal(b, latestBlockHash) {
				newInTransit = append(newInTransit, b)
			}
		}

		BlockInTransit = newInTransit // 更新BlockInTransit变量，移除已经请求过的区块hash
	}
}

type GetData struct {
	AddrFrom string // 请求方的地址
	Type     string // 请求的数据类型
	ID       []byte // 请求的区块hash
}

// getBlockData gets block data from a node
//
// 向toAddr发送getdata命令，请求对应区块的数据
func getBlockData(toAddr, kind string, blockHash []byte) {
	// 1. 构建getdata命令, 并且转化成字节数组
	payload := EncodeEverything(GetData{AddrFrom: CurrentNode, Type: kind, ID: blockHash})
	request := append(commandToBytes("getdata"), payload...)

	// 2. 向toAddr发送getdata命令
	sendData(toAddr, request)
}

// handleGetData handles getdata message from other nodes
//
// 处理其他节点发送过来的getdata命令，根据区块hash，获取对应的区块数据，然后发送给请求方
func handleGetData(request []byte, bc *Blockchain) {
	// 1. 把request 字节切片转化成Getdata结构体
	var buff bytes.Buffer
	var payload GetData

	decoder := gob.NewDecoder(&buff)
	buff.Write(request[COMMANDLENGTH:])
	err := decoder.Decode(&payload)
	if err != nil {
		panic(err)
	}

	// 2. 根据区块的hash，获取对应的区块数据
	if payload.Type == "block" {
		block, err := bc.GetBlock(payload.ID) // 根据区块hash，获取对应的区块数据
		if err != nil {
			panic(err)
		}

		// 找到了区块数据，就把区块数据发送给请求方
		sendBlock(payload.AddrFrom, &block)
	}

}

// GetBlock returns a block by its hash
// TODO: move to blockchain.go
// 根据区块hash，获取对应的区块数据
func (bc *Blockchain) GetBlock(blockHash []byte) (Block, error) {
	var block Block

	db := bc.db

	// 1. 获取区块数据
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(BLOCKBUCKET))
		blockData := b.Get(blockHash) // 根据区块hash，获取对应的区块数据

		// 如果没有找到区块数据，返回错误
		if blockData == nil {
			return fmt.Errorf("Block is not found")
		}
		// 2. 反序列化区块数据
		block = *Deserialize(blockData)
		return nil
	})

	if err != nil {
		return block, err
	}

	return block, nil
}

type SendBlock struct {
	AddrFrom string
	Block    []byte
}

// sendBlock sends block message to a node
//
// 把区块数据发送给请求方
func sendBlock(toAddr string, block *Block) {
	// 1. 构建block命令, 并且转化成字节数组
	payload := EncodeEverything(SendBlock{AddrFrom: CurrentNode, Block: block.Serialize()})
	request := append(commandToBytes("block"), payload...)

	// 2. 向toAddr发送block命令
	sendData(toAddr, request)
}

// handleBlock handles block message from other nodes
//
// 处理其他节点发送过来的block命令，把区块添加到本地区块链中，并且更新UTXO集合
func handleBlock(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload SendBlock

	// 1. 把request 字节切片转化成SendBlock结构体
	decoder := gob.NewDecoder(&buff)
	buff.Write(request[COMMANDLENGTH:])
	err := decoder.Decode(&payload)
	if err != nil {
		panic(err)
	}

	// 2. 反序列化区块数据
	block := Deserialize(payload.Block)

	// 3. 把区块添加到区块链中
	bc.AddBlockBy(block) // 把区块添加到区块链中
	fmt.Printf("Received a new block and add it to blockchain! %v", block)

	if len(BlockInTransit) > 0 {
		blockHash := BlockInTransit[0]                     // 获取第一个区块hash
		getBlockData(payload.AddrFrom, "block", blockHash) // 根据区块hash，获取对应的区块数据
		BlockInTransit = BlockInTransit[1:]                // 移除第一个区块hash
	} else {
		// update UTXO set
		utxoSet := UTXOSet{bc}
		utxoSet.UpdateUTXO(block) // 使用新区块更新UTXO集合
	}
}

// AddBlockBy adds a block to the blockchain
//
// 把区块添加到区块链中
func (bc *Blockchain) AddBlockBy(block *Block) {
	// 1. 把区块添加到区块链数据库中
	err := bc.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BLOCKBUCKET))
		// 把区块数据添加到区块链数据库中
		err := bucket.Put(block.Hash, block.Serialize())
		if err != nil {
			panic(err)
		}

		// update the latest block if necessary
		latestHash := bucket.Get([]byte("latest"))
		latestBlockData := bucket.Get(latestHash)
		latestBlock := Deserialize(latestBlockData)

		// 对比区块高度，如果新区块的高度大于最新区块的高度，就更新最新区块的hash
		if block.Height > latestBlock.Height {
			err = bucket.Put([]byte("latest"), block.Hash)
			if err != nil {
				panic(err)
			}
			bc.topHash = block.Hash // 更新区块链最新区块的hash
		}

		return nil
	})

	if err != nil {
		panic(err)
	}
}
