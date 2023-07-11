package main

import (
	"flag"
	"fmt"
	"os"
)

// CLI responsible for processing command line arguments

type CLI struct {
	blockchain *Blockchain
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

	// create a flagset addblock for addblock command,
	// which
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

	switch os.Args[1] {
	//如果第一个命令行参数等于 "addblock"，那么就执行下面的代码块。
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

}

// addBlock add a new block to the blockchain using CLI
//
// 使用CLI添加一个新的区块到区块链
func (cli *CLI) addBlock() {

	// new a slice of random transactions
	txs := []*Transaction{
		CoinBaseTx("alice"),
		CoinBaseTx("bob"),
	}
	cli.blockchain.AddBlock(txs)
}

func (cli *CLI) printBlock() {
	cli.blockchain.IterateBlockchain()
}

func (cli *CLI) GetBalance(addr string) {
	balance := 0
	utxos := cli.blockchain.FindUTXO(addr)

	for _, utxo := range utxos {
		balance += utxo.Value
	}

	fmt.Printf("Balance of %s: %d\n", addr, balance)
}

func (cli *CLI) SendTx(from, to string, amount int) {
	tx := CreateTransaction(from, to, amount, cli.blockchain)

	cli.blockchain.AddBlock([]*Transaction{tx})

	fmt.Println("Success!")
}
