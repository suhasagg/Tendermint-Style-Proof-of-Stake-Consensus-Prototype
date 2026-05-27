package main

import (
	"fmt"

	"tendermint-pos-prototype/consensus"
	"tendermint-pos-prototype/types"
)

func main() {
	engine := consensus.NewEngine([]types.Validator{
		{Address: "val1", Power: 40},
		{Address: "val2", Power: 30},
		{Address: "val3", Power: 20},
		{Address: "val4", Power: 10},
	})

	engine.Mempool.Add("tx-1: alice sends 10 tokens to bob")
	engine.Mempool.Add("tx-2: carol stakes 50 tokens")

	block, commit, err := engine.RunHeight(0)
	if err != nil {
		panic(err)
	}

	fmt.Println("Committed block")
	fmt.Println("height:", block.Height)
	fmt.Println("proposer:", block.Proposer)
	fmt.Println("hash:", block.Hash())
	fmt.Println("commit votes:", len(commit.Votes))
}
