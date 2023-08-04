package main

import "fmt"

func main() {
	// test()

	cli := CLI{}
	cli.Blockchain = CreateBlockchain()
	cli.Run()

}

func test() {

	// 测试base	58编码解码是否正常

	// new wallet
	wallet := CreateWallet()

	fmt.Printf("private key: %x\n", wallet.PrivateKey.D.Bytes())
	fmt.Printf("public key: %x\n", wallet.PublicKey)
	address := wallet.GetAddressWithPublickey(MAINNET_VERSION)
	fmt.Printf("address: %x\n", address)

	ok := ValidateAddress(string(address))

	fmt.Printf("validate address: %v\n", ok)
}
