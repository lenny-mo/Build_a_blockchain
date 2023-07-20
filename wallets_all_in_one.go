package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

const (
	WALLETSFILE = "wallets.dat"
)

type Wallets struct {
	Wallets map[string]*Wallet // map[address]*Wallet
}

// CreateWallets creates a new wallets to store a number of wallets
//
// 创建一个新的钱包集合
func CreateWallets() *Wallets {
	ws := &Wallets{}
	ws.Wallets = make(map[string]*Wallet) // 初始化map, 任何对nil map的操作都会引发panic
	return ws
}

// CreateWalletRandomly creates a wallet randomly
//
// 随机创建一个钱包并且保存到mapping中, 返回钱包的公钥地址
func (ws *Wallets) CreateWalletRandomly() string {
	wallet := CreateWallet()
	address := wallet.GetAddressWithPublickey(MAINNET_VERSION)
	ws.Wallets[string(address)] = wallet

	return string(address)
}

func (ws *Wallets) GetWallet(address string) *Wallet {
	return ws.Wallets[address]
}

func (ws *Wallets) getAllAddress() []string {
	var addresses []string
	// iterate over all keys in the map
	for address := range ws.Wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

// SaveWalletsToFile saves wallets to file
//
// 将钱包保存到文件中
func (ws *Wallets) SaveWalletsToFile() bool {
	// Go 语言标准库中的一个类型，它是一个可以读写的字节缓冲区。你可以向这个缓冲区写入字节，也可以从这个缓冲区读取字节
	var buffer bytes.Buffer
	gob.Register(secp256k1.S256()) // 注册椭圆曲线
	// 将数据编码为 gob 格式，也就是binary格式, 编码后的数据会被写入到 buffer 中
	encoder := gob.NewEncoder(&buffer)
	// 执行编码操作, 传递结构体指针避免传递大结构体
	err := encoder.Encode(ws)
	if err != nil {
		fmt.Printf("create encoder failed while saving wallet to file : %v\n", err)
		return false
	}

	// 将 buffer 中的数据以追加的形式写入到文件中，0666 表示文件所有用户可读可写
	// os.O_CREATE模式表示如果文件不存在，那么会创建一个新的文件。
	// os.O_WRONLY模式表示文件被打开以供写入数据，不能用于读取数据。
	file, err := os.OpenFile(WALLETSFILE, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("open wallets file failed: %v\n", err)
		return false
	}
	defer file.Close()

	// 将 buffer 中的数据写入到文件中
	_, err = file.Write(buffer.Bytes())
	if err != nil {
		fmt.Printf("write wallets file failed: %v\n", err)
		return false
	}

	return true
}

// ReadWalletsFromFile reads wallets from file
//
// 从文件中读取钱包
func (ws *Wallets) ReadWalletsFromFile() bool {
	// 判断文件是否存在
	if _, err := os.Stat(WALLETSFILE); os.IsNotExist(err) {
		fmt.Printf("wallets file doesn't exist! we will create a new one!\n")
		return false
	}

	// 读取文件中的数据
	file, err := os.Open(WALLETSFILE)
	if err != nil {
		fmt.Printf("open wallets file failed: %v\n", err)
		return false
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("read wallets file failed: %v\n", err)
		return false
	}

	// 反序列化
	var wallets Wallets                              // 创建一个临时的wallets
	gob.Register(secp256k1.S256())                   // 注册椭圆曲线
	decoder := gob.NewDecoder(bytes.NewReader(data)) // 创建一个解码器
	err = decoder.Decode(&wallets)                   // 把解码后的数据放到wallets中
	if err != nil {
		fmt.Printf("decode wallets file failed: %v\n", err)
		return false
	}

	// 所以当你把一个map赋值给另一个map，你其实是创建了一个新的引用（或者指针），
	// 它指向的是原来的map。因此，如果你改变其中一个map，另一个也会发生改变，因为它们都指向同一块内存空间
	ws.Wallets = wallets.Wallets // 把解码后的数据放到当前的wallets中, 这里的ws是指针，所以可以直接赋值

	return true
}
