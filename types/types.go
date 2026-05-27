package types

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type Address string

type Validator struct {
	Address Address
	Power   int64
	Jailed  bool
}

type ValidatorSet struct {
	Validators []Validator
}

func NewValidatorSet(vals []Validator) *ValidatorSet {
	cp := append([]Validator(nil), vals...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Address < cp[j].Address })
	return &ValidatorSet{Validators: cp}
}

func (vs *ValidatorSet) TotalPower() int64 {
	var total int64
	for _, v := range vs.Validators {
		if !v.Jailed {
			total += v.Power
		}
	}
	return total
}

func (vs *ValidatorSet) PowerOf(addr Address) int64 {
	for _, v := range vs.Validators {
		if v.Address == addr && !v.Jailed {
			return v.Power
		}
	}
	return 0
}

func (vs *ValidatorSet) Proposer(height int64, round int) Validator {
	active := make([]Validator, 0)
	for _, v := range vs.Validators {
		if !v.Jailed && v.Power > 0 {
			active = append(active, v)
		}
	}
	if len(active) == 0 {
		return Validator{}
	}
	idx := int((height + int64(round)) % int64(len(active)))
	return active[idx]
}

func (vs *ValidatorSet) Jail(addr Address) {
	for i := range vs.Validators {
		if vs.Validators[i].Address == addr {
			vs.Validators[i].Jailed = true
		}
	}
}

type Tx string

type Block struct {
	Height       int64
	Round        int
	Proposer     Address
	Txs          []Tx
	PreviousHash string
}

func (b Block) Hash() string {
	txs := make([]string, len(b.Txs))
	for i, tx := range b.Txs {
		txs[i] = string(tx)
	}
	raw := fmt.Sprintf("%d|%d|%s|%s|%s", b.Height, b.Round, b.Proposer, b.PreviousHash, strings.Join(txs, ","))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

type VoteType string

const (
	Prevote   VoteType = "PREVOTE"
	Precommit VoteType = "PRECOMMIT"
)

type Vote struct {
	Type      VoteType
	Height    int64
	Round     int
	Validator Address
	BlockHash string
	Signature string
}

func NewVote(t VoteType, height int64, round int, validator Address, blockHash string) Vote {
	payload := fmt.Sprintf("%s|%d|%d|%s|%s", t, height, round, validator, blockHash)
	sum := sha256.Sum256([]byte(payload))
	return Vote{Type: t, Height: height, Round: round, Validator: validator, BlockHash: blockHash, Signature: hex.EncodeToString(sum[:])}
}

func (v Vote) Verify() bool {
	expected := NewVote(v.Type, v.Height, v.Round, v.Validator, v.BlockHash)
	return expected.Signature == v.Signature
}

type Commit struct {
	Height    int64
	Round     int
	BlockHash string
	Votes     []Vote
}
