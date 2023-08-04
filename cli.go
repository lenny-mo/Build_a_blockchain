package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// CLI responsible for processing command line arguments

type CLI struct {
	Blockchain *Blockchain
}

// validateArgs validates the arguments of the command line, must be greater than 1
//
// 验证命令行参数，必须大于1, os.Args[0]是程序名, os.Args[1]是第一个参数
func (cli *CLI) validateArgs() {
	// cli arg len must be greater than 1
	if len(os.Args) < 2 {
		fmt.Println("cli arg len must be greater than 1!")
		os.Exit(1)
	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	// --------------------- 1. create a flagset addblock for addblock command ---------------------
	startNode := flag.NewFlagSet("startnode", flag.ExitOnError)
	startNodeMinner := startNode.String("minner", "", "Start a node with miner address")

	addBlock := flag.NewFlagSet("addblock", flag.ExitOnError)
	printBlock := flag.NewFlagSet("printblock", flag.ExitOnError)
	getBalance := flag.NewFlagSet("getbalance", flag.ExitOnError)
	// 在 "getbalance" 这个 FlagSet 对象中定义了一个新的字符串参数 "address"。
	// 可以通过 -address 参数来提供一个地址
	// default value is "", usage is "The address to get balance for"
	addr := getBalance.String("address", "", "The address to get balance for")

	sendtx := flag.NewFlagSet("sendtx", flag.ExitOnError)
	sendtxFrom := sendtx.String("from", "", "Source wallet address")
	sendtxTo := sendtx.String("to", "", "Destination wallet address")
	sendtxAmount := sendtx.Int("amount", 0, "Amount to send")

	// 创建钱包
	createWallet := flag.NewFlagSet("createwallet", flag.ExitOnError)
	listAddress := flag.NewFlagSet("listaddress", flag.ExitOnError)

	// 获取最新区块高度
	getLatestHeight := flag.NewFlagSet("getlatestheight", flag.ExitOnError)

	// -------------------------- 2. 解析命令行参数 --------------------------
	// os.Args[0]是程序的路径, os.Args[1]是第一个参数
	switch os.Args[1] {

	case "startnode":
		err := startNode.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	case "getlatestheight":
		err := getLatestHeight.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}
	// 打印wallets.dat中的所有地址
	case "listaddress":
		err := listAddress.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	// 创建钱包并且保存到wallets.dat中
	case "createwallet":
		err := createWallet.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	case "addblock":
		// 解析从第二个参数开始的所有命令行参数, 把命令行参数转换成程序可以使用的数据和配置
		err := addBlock.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	case "printblock":
		err := printBlock.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	case "getbalance":
		err := getBalance.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	case "sendtx":
		err := sendtx.Parse(os.Args[2:])
		if err != nil {
			panic(err)
		}

	default:
		fmt.Println("invalid command")
		os.Exit(1)
	}

	// ------------------------ 3. 根据解析后的命令行参数执行对应的功能 ------------------------
	if startNode.Parsed() {
		nodeID := "3000" // 需要监听的节点ID
		if nodeID == "" {
			fmt.Println("NODE_ID env. var is not set! You need to set it to a positive integer value")
			os.Exit(1)
		}
		cli.startnode(nodeID, *startNodeMinner)
	}

	if getLatestHeight.Parsed() {
		cli.GetLatestHeight()
	}

	if addBlock.Parsed() {
		cli.addBlock()
	}

	if printBlock.Parsed() {
		cli.printBlock()
	}

	if getBalance.Parsed() {
		// check if the address is valid
		if len(os.Args[2]) == 0 {
			fmt.Println("invalid address")
			os.Exit(1)
		}
		cli.GetBalance(*addr)
	}

	if sendtx.Parsed() {
		if len(*sendtxFrom) == 0 {
			fmt.Println("invalid sender address")
			os.Exit(1)
		}
		if len(*sendtxTo) == 0 {
			fmt.Println("invalid receiver address")
			os.Exit(1)
		}
		if *sendtxAmount <= 0 {
			fmt.Println("invalid amount")
			os.Exit(1)
		}

		cli.SendTx(*sendtxFrom, *sendtxTo, *sendtxAmount)
	}

	if createWallet.Parsed() {
		cli.CreateWallet()
	}

	if listAddress.Parsed() {
		cli.ListAddress()
	}

}

// addBlock add a new block to the blockchain using CLI
//
// 使用CLI添加一个新的区块到区块链
func (cli *CLI) addBlock() {

	// new a slice of random transactions
	txs := []*Transaction{
		CoinBaseTx("1FBae9FyJTofCbWYK2hMHnxtf78qreFTSD"),
	}
	cli.Blockchain.AddBlock(txs)
}

func (cli *CLI) printBlock() {
	cli.Blockchain.IterateBlockchain()
}

// GetBalance get the balance of the address
func (cli *CLI) GetBalance(addr string) {
	balance := 0
	// utxos := cli.Blockchain.FindUTXO(AddressToPubkeyHash(addr))

	utxoset := UTXOSet{cli.Blockchain}
	utxoset.StoreUTXO()
	utxos := utxoset.FindUTXOByPubkeyHash(AddressToPubkeyHash(addr))

	for _, utxo := range utxos {
		balance += utxo.Value
	}

	fmt.Printf("Balance of %s: %d\n", addr, balance)
}

func (cli *CLI) SendTx(from, to string, amount int) {
	tx := CreateTransaction(from, to, amount, cli.Blockchain)

	_, newblock := cli.Blockchain.AddBlock([]*Transaction{tx})

	utsoxet := UTXOSet{cli.Blockchain}

	utsoxet.UpdateUTXO(newblock)

	// 硬编码形式验证UpdateUTXO是否正确
	cli.GetBalance("1FBae9FyJTofCbWYK2hMHnxtf78qreFTSD")
	cli.GetBalance("13bkBrCPM8tiCaXNufbWcQnRBjTboxK9jM")
	cli.GetBalance("1D5S4w2ApAwhBgnYrRhtQx4X2J9uwSDCaD")
	cli.GetBalance("1MFHDXugCGge6wxAhoS5Pee2AoVnpyfC7L")
	cli.GetBalance("14AcsbEULnBTSU44HV8wswH12bmuJ78b9G")

	fmt.Println("Success!")
}

func (cli *CLI) CreateWallet() {
	wallets := CreateWallets()
	wallets.ReadWalletsFromFile()             // 读取已经存在的钱包
	address := wallets.CreateWalletRandomly() // 添加一个新的钱包
	wallets.SaveWalletsToFile()               // 再次保存到文件中

	fmt.Println("Your new address: ", address)
}

func (cli *CLI) ListAddress() {
	wallets := CreateWallets()
	addresses := wallets.getAllAddress()

	for _, address := range addresses {
		fmt.Printf("address: %s\n", address)
	}
}

func (cli *CLI) GetLatestHeight() {
	height, _ := cli.Blockchain.GetLatestHeight()
	fmt.Printf("latest height: %d\n", height)
}

// startnode start a node
func (cli CLI) startnode(nodeid, minnerAddr string) {
	fmt.Printf("start node: %s\n", nodeid)

	// 如果minnerAddr不为空，则验证地址是否有效
	if len(minnerAddr) > 0 {
		if ValidateAddress(minnerAddr) {
			fmt.Println("minner address is valid!")
		} else {
			log.Panic("invalid minner address")
		}
	}

	// 如果节点有效，则启动服务器
	ok := StartServer(nodeid, minnerAddr, cli.Blockchain)
	if !ok {
		fmt.Printf("node %s start failed!\n", nodeid)
		os.Exit(1)
	}

}
