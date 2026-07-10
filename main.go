package main

import (
	"flag"
	"fmt"

	"toyblockchain/chain"
	"toyblockchain/cli"
)

func main() {

	// FR-9: Configurable parameters.

	flag.IntVar(
		&chain.DefaultDifficulty,
		"difficulty",
		chain.DefaultDifficulty,
		"Mining difficulty (leading zeroes)",
	)

	flag.IntVar(
		&chain.DefaultBlockSize,
		"blocksize",
		chain.DefaultBlockSize,
		"Maximum transactions per block",
	)

	flag.StringVar(
		&chain.DefaultBlockchainFile,
		"data",
		chain.DefaultBlockchainFile,
		"Blockchain data file",
	)

	flag.Parse()

	// Display active configuration

	fmt.Println("===================================")
	fmt.Println("Toy Blockchain Configuration")
	fmt.Println("===================================")

	fmt.Println(
		"Difficulty :",
		chain.DefaultDifficulty,
	)

	fmt.Println(
		"Block Size :",
		chain.DefaultBlockSize,
	)

	fmt.Println(
		"Data File  :",
		chain.DefaultBlockchainFile,
	)

	fmt.Println()

	cli.Run()
}
