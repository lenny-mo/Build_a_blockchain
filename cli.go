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

	// print all args including args[0]
	for i, arg := range os.Args {
		fmt.Printf("Args[%d]: %v", i, arg)

	}
}

func (cli *CLI) Run() {
	cli.validateArgs()

	// create a flagset addblock for addblock command,
	// which
	addBlock := flag.NewFlagSet("addblock", flag.ExitOnError)
	printBlock := flag.NewFlagSet("printblock", flag.ExitOnError)

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
