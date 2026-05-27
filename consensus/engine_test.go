package consensus

import (
	"testing"

	"tendermint-pos-prototype/types"
)

func validators() []types.Validator {
	return []types.Validator{
		{Address: "val1", Power: 40},
		{Address: "val2", Power: 30},
		{Address: "val3", Power: 20},
		{Address: "val4", Power: 10},
	}
}

func TestRunHeightCommitsBlock(t *testing.T) {
	e := NewEngine(validators())
	e.Mempool.Add("alice->bob:10")
	e.Mempool.Add("carol->dan:5")

	block, commit, err := e.RunHeight(0)
	if err != nil {
		t.Fatalf("RunHeight error: %v", err)
	}
	if block.Height != 1 {
		t.Fatalf("height got %d", block.Height)
	}
	if commit.BlockHash != block.Hash() {
		t.Fatalf("commit hash mismatch")
	}
	if e.Chain.Height() != 1 {
		t.Fatalf("chain height got %d", e.Chain.Height())
	}
	if len(block.Txs) != 2 {
		t.Fatalf("tx count got %d", len(block.Txs))
	}
}

func TestTwoThirdsWeightedVoting(t *testing.T) {
	e := NewEngine(validators())
	hash := "block-a"
	votes := []types.Vote{
		types.NewVote(types.Prevote, 1, 0, "val1", hash),
		types.NewVote(types.Prevote, 1, 0, "val2", hash),
	}
	if !e.HasTwoThirds(votes, hash) {
		t.Fatalf("expected 70/100 to pass +2/3")
	}

	votes = []types.Vote{
		types.NewVote(types.Prevote, 1, 0, "val1", hash),
		types.NewVote(types.Prevote, 1, 0, "val3", hash),
	}
	if e.HasTwoThirds(votes, hash) {
		t.Fatalf("expected 60/100 to fail +2/3")
	}
}

func TestDoubleSignSlashesValidator(t *testing.T) {
	e := NewEngine(validators())
	votes := []types.Vote{
		types.NewVote(types.Prevote, 2, 0, "val1", "hash-a"),
		types.NewVote(types.Prevote, 2, 0, "val1", "hash-b"),
	}
	slashed := e.DetectAndSlashDoubleSign(votes)
	if len(slashed) != 1 || slashed[0] != "val1" {
		t.Fatalf("expected val1 slashed, got %v", slashed)
	}
	if e.Validators.PowerOf("val1") != 0 {
		t.Fatalf("jailed validator should have zero active power")
	}
}

func TestInvalidSignatureIgnored(t *testing.T) {
	e := NewEngine(validators())
	hash := "block-a"
	v := types.NewVote(types.Prevote, 1, 0, "val1", hash)
	v.Signature = "tampered"
	votes := []types.Vote{v, types.NewVote(types.Prevote, 1, 0, "val2", hash), types.NewVote(types.Prevote, 1, 0, "val3", hash)}
	if e.HasTwoThirds(votes, hash) {
		t.Fatalf("tampered vote must be ignored")
	}
}
