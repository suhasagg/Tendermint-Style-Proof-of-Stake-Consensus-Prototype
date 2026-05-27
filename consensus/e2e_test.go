package consensus

import (
	"errors"
	"testing"

	"tendermint-pos-prototype/types"
)

func e2eValidators() []types.Validator {
	return []types.Validator{
		{Address: "alice-validator", Power: 40},
		{Address: "bob-validator", Power: 30},
		{Address: "carol-validator", Power: 20},
		{Address: "dan-validator", Power: 10},
	}
}

func TestE2EMultipleHeightsCommitAndDrainMempool(t *testing.T) {
	engine := NewEngine(e2eValidators())
	engine.MaxTxs = 2

	engine.Mempool.Add("tx-1: alice->bob:10")
	engine.Mempool.Add("tx-2: bob->carol:5")
	engine.Mempool.Add("tx-3: carol stakes:100")
	engine.Mempool.Add("tx-4: dan votes proposal-7")
	engine.Mempool.Add("tx-5: alice unstakes:20")

	block1, commit1, err := engine.RunHeight(0)
	if err != nil {
		t.Fatalf("height 1 should commit: %v", err)
	}
	if block1.Height != 1 || commit1.Height != 1 {
		t.Fatalf("height 1 mismatch: block=%d commit=%d", block1.Height, commit1.Height)
	}
	if len(block1.Txs) != 2 {
		t.Fatalf("height 1 should include 2 txs, got %d", len(block1.Txs))
	}
	if commit1.BlockHash != block1.Hash() {
		t.Fatalf("height 1 commit hash mismatch")
	}

	block2, commit2, err := engine.RunHeight(0)
	if err != nil {
		t.Fatalf("height 2 should commit: %v", err)
	}
	if block2.Height != 2 || commit2.Height != 2 {
		t.Fatalf("height 2 mismatch: block=%d commit=%d", block2.Height, commit2.Height)
	}
	if block2.PreviousHash != block1.Hash() {
		t.Fatalf("height 2 previous hash should point to height 1")
	}
	if len(block2.Txs) != 2 {
		t.Fatalf("height 2 should include 2 txs, got %d", len(block2.Txs))
	}

	block3, commit3, err := engine.RunHeight(0)
	if err != nil {
		t.Fatalf("height 3 should commit: %v", err)
	}
	if block3.Height != 3 || commit3.Height != 3 {
		t.Fatalf("height 3 mismatch: block=%d commit=%d", block3.Height, commit3.Height)
	}
	if block3.PreviousHash != block2.Hash() {
		t.Fatalf("height 3 previous hash should point to height 2")
	}
	if len(block3.Txs) != 1 {
		t.Fatalf("height 3 should include remaining 1 tx, got %d", len(block3.Txs))
	}

	if engine.Chain.Height() != 3 {
		t.Fatalf("chain should be at height 3, got %d", engine.Chain.Height())
	}
	if len(engine.Mempool.txs) != 0 {
		t.Fatalf("mempool should be empty after three heights, got %d", len(engine.Mempool.txs))
	}
}

func TestE2EDoubleSignSlashingAffectsFutureConsensus(t *testing.T) {
	engine := NewEngine(e2eValidators())

	conflictingVotes := []types.Vote{
		types.NewVote(types.Precommit, 1, 0, "alice-validator", "block-hash-a"),
		types.NewVote(types.Precommit, 1, 0, "alice-validator", "block-hash-b"),
		types.NewVote(types.Precommit, 1, 0, "bob-validator", "block-hash-a"),
	}

	slashed := engine.DetectAndSlashDoubleSign(conflictingVotes)
	if len(slashed) != 1 || slashed[0] != "alice-validator" {
		t.Fatalf("expected alice-validator to be slashed, got %v", slashed)
	}
	if engine.Validators.TotalPower() != 60 {
		t.Fatalf("active power after slashing should be 60, got %d", engine.Validators.TotalPower())
	}

	engine.Mempool.Add("tx-after-slash")
	block, commit, err := engine.RunHeight(0)
	if err != nil {
		t.Fatalf("remaining honest validators should still commit: %v", err)
	}
	if block.Height != 1 || commit.Height != 1 {
		t.Fatalf("expected first committed height after slashing to be 1")
	}
	for _, vote := range commit.Votes {
		if vote.Validator == "alice-validator" {
			t.Fatalf("jailed validator must not appear in future commit votes")
		}
	}
}

func TestE2ENoActiveValidatorsFailsToCommit(t *testing.T) {
	engine := NewEngine(e2eValidators())
	for _, validator := range e2eValidators() {
		engine.Validators.Jail(validator.Address)
	}

	_, _, err := engine.RunHeight(0)
	if !errors.Is(err, ErrNoActiveValidators) {
		t.Fatalf("expected ErrNoActiveValidators, got %v", err)
	}
	if engine.Chain.Height() != 0 {
		t.Fatalf("chain must stay at genesis height on failure, got %d", engine.Chain.Height())
	}
}
