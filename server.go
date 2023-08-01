package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
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
	KNOWNNODES  = []string{"localhost:3000"} // 种子节点列表
	CURRENTNODE = ""                         // 当前节点
)

// StartServer starts a node
func StartServer(nodeID, minderAddr string) bool {
	nodeAddr := fmt.Sprintf("localhost:%s", nodeID)
	CURRENTNODE = nodeAddr

	listener, err := net.Listen("tcp", nodeAddr) // 监听这个节点的地址
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	blockchain := CreateBlockchain()

	// 如果当前节点不是种子节点，那么就需要向种子节点发送版本信息
	if nodeAddr != KNOWNNODES[0] {
		sendVersion(nodeAddr, blockchain)
	}

	// 一直监听
	for {
		connect, err := listener.Accept() // 接收到一个连接
		if err != nil {
			panic(err)
		}
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
		CURRENTNODE, // 发送方地址: 当前节点的地址
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
// 把一个命令转换成长度固定的字节数组, 考虑是否可以使用hash, 把命令转换成固定长度的哈希值
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
	var command = make([]byte, 0, 0)

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
// 向一个节点发送数据, 如果节点不存在, 那么就把这个节点从种子节点列表中删除
func sendData(toAddr string, data []byte) bool {
	connect, err := net.Dial("tcp", toAddr)
	// 如果这个地址无法连接，那么就把这个地址从种子节点列表中删除
	if err != nil {
		fmt.Printf("address %s is not available\n", toAddr)

		updateNodes := []string{}

		// 遍历种子节点列表，把无法连接的节点剔除
		// updateNodes列表中就包含了所有可以连接的节点，而无法连接的节点（即toAddr）被排除在外。
		for _, node := range KNOWNNODES {
			if node != toAddr {
				updateNodes = append(updateNodes, node)
			}
		}

		KNOWNNODES = updateNodes // 返回剔除之后的种子节点列表
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
	request, err := io.ReadAll(conn)
	if err != nil {
		panic(err)
	}

	// 从request中解析出命令
	command := bytesToCommand(request[:COMMANDLENGTH])

	// 执行对应的函数
	switch command {
	case "version":
		handleVersion(request, bc)
	}

}

// handleVersion handles version message
//
// 处理version信息
func handleVersion(request []byte, bc *Blockchain) {
	var buff bytes.Buffer
	var payload Version // 指代在一个数据包或消息中，实际携带的、对于最终用户有意义的数据

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

	localHeight, _ := bc.GetLatestHeight() // 获取当前节点的区块高度
	remoteHeight := payload.LatestHeight   // 获取发送方的区块高度

	if remoteHeight > localHeight {
		// 如果发送方的区块高度大于当前节点的区块高度，那么就向发送方请求区块数据
		// TODO: getBlocksFrom()
	} else if remoteHeight < localHeight {
		// 如果发送方的区块高度小于当前节点的区块高度，那么就向发送方发送区块数据
		sendVersion(payload.Addrfrom, bc)
	}

	// 如果发送方的地址不在种子节点列表中，那么就把发送方的地址添加到种子节点列表中
	if !isKnownNode(payload.Addrfrom) {
		KNOWNNODES = append(KNOWNNODES, payload.Addrfrom)
	}

}

// isKnownNode checks if a node is known
//
// 判断一个节点是否在种子节点列表中
func isKnownNode(addr string) bool {
	for _, node := range KNOWNNODES {
		if node == addr {
			return true
		}
	}

	return false
}
