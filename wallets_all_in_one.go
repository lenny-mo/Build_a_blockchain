package main

import (
	"bytes"
	"encoding/gob"
)

const (
	walletsFile = "wallets.dat"
)

type Walleets struct {
	wallets map[string]*Wallet // map[address]*Wallet
}

func CreateWalletsFile() *Walleets {
	ws := &Walleets{}
	ws.wallets = make(map[string]*Wallet)
	return ws
}

// CreateWalletRandomly creates a wallet randomly
//
// 随机创建一个钱包并且保存到mapping中
func (ws *Walleets) CreateWalletRandomly() string {
	wallet := CreateWallet()
	address := wallet.GetAddressWithPublickey(MAINNET_VERSION)
	ws.wallets[string(address)] = wallet

	return string(address)
}

func (ws *Walleets) GetWallet(address string) *Wallet {
	return ws.wallets[address]
}

func (ws *Walleets) getAllAddress() []string {
	var addresses []string
	// iterate over all keys in the map
	for address := range ws.wallets {
		addresses = append(addresses, address)
	}

	return addresses
}

func (ws *Walleets) SaveWalletsToFile() bool {
	var buffer bytes.Buffer
	
	encoder := gob.NewEncoder(&buffer)
}
