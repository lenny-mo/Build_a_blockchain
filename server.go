package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
)

type Version struct {
	Version      int
	LatestHeight int64
	Addrfrom     string // 发送方地址
}

const (
	KNOWNNODES  = "localhost:3000"
	NODEVERSION = 1
)

// StartServer starts a node
func StartServer(nodeID, minderAddr string) bool {
	nodeAddr := fmt.Sprintf("localhost:%s", nodeID)
	listener, err := net.Listen("tcp", nodeAddr) // 监听这个节点的地址
	if err != nil {
		panic(err)
		return false
	}
	defer listener.Close()

	blockchain := CreateBlockchain()

	// 如果当前节点不是种子节点，那么就需要向种子节点发送版本信息
	if nodeAddr != KNOWNNODES {
		sendVersion(nodeAddr, KNOWNNODES, blockchain)
	}

	//
	for {
		connect, err := listener.Accept() // 接收到一个连接
		if err != nil {
			panic(err)
		}
		go handleConnection(connect, blockchain)
	}

}

func sendVersion(to, from string, bc *Blockchain) {
	latestHeight, _ := bc.GetLatestHeight()

	version := Version{
		NODEVERSION,
		latestHeight,
		from,
	}

	payload := EncodeEverything(version)
	request := append([]byte("version"), payload...)
	sendData(to, request)
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
