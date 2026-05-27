package consensus

import (
	"errors"
	"fmt"

	"tendermint-pos-prototype/types"
)

var (
	ErrNoActiveValidators = errors.New("no active validators")
	ErrNoTwoThirds        = errors.New("failed to reach +2/3 voting power")
	ErrInvalidVote        = errors.New("invalid vote")
)

type Mempool struct {
	txs []types.Tx
}

func NewMempool() *Mempool { return &Mempool{} }

func (m *Mempool) Add(tx types.Tx) { m.txs = append(m.txs, tx) }

func (m *Mempool) Drain(max int) []types.Tx {
	if max <= 0 || max > len(m.txs) {
		max = len(m.txs)
	}
	out := append([]types.Tx(nil), m.txs[:max]...)
	m.txs = m.txs[max:]
	return out
}

type Blockchain struct {
	Blocks []types.Block
}

func NewBlockchain() *Blockchain {
	genesis := types.Block{Height: 0, Round: 0, Proposer: "genesis", PreviousHash: "", Txs: nil}
	return &Blockchain{Blocks: []types.Block{genesis}}
}

func (bc *Blockchain) Height() int64 { return int64(len(bc.Blocks) - 1) }

func (bc *Blockchain) LastHash() string { return bc.Blocks[len(bc.Blocks)-1].Hash() }

func (bc *Blockchain) Append(block types.Block) error {
	if block.Height != bc.Height()+1 {
		return fmt.Errorf("invalid height: got %d want %d", block.Height, bc.Height()+1)
	}
	if block.PreviousHash != bc.LastHash() {
		return fmt.Errorf("invalid previous hash")
	}
	bc.Blocks = append(bc.Blocks, block)
	return nil
}

type EvidencePool struct {
	DoubleSigns []types.Address
}

func (e *EvidencePool) AddDoubleSign(addr types.Address) { e.DoubleSigns = append(e.DoubleSigns, addr) }

type Engine struct {
	Validators *types.ValidatorSet
	Mempool    *Mempool
	Chain      *Blockchain
	Evidence   *EvidencePool
	MaxTxs     int
}

func NewEngine(vals []types.Validator) *Engine {
	return &Engine{
		Validators: types.NewValidatorSet(vals),
		Mempool:    NewMempool(),
		Chain:      NewBlockchain(),
		Evidence:   &EvidencePool{},
		MaxTxs:     100,
	}
}

func (e *Engine) RunHeight(round int) (types.Block, types.Commit, error) {
	height := e.Chain.Height() + 1
	proposer := e.Validators.Proposer(height, round)
	if proposer.Address == "" {
		return types.Block{}, types.Commit{}, ErrNoActiveValidators
	}

	block := types.Block{
		Height:       height,
		Round:        round,
		Proposer:     proposer.Address,
		PreviousHash: e.Chain.LastHash(),
		Txs:          e.Mempool.Drain(e.MaxTxs),
	}

	prevotes := e.collectVotes(types.Prevote, height, round, block.Hash())
	if !e.hasTwoThirds(prevotes, block.Hash()) {
		return block, types.Commit{}, ErrNoTwoThirds
	}

	precommits := e.collectVotes(types.Precommit, height, round, block.Hash())
	if !e.hasTwoThirds(precommits, block.Hash()) {
		return block, types.Commit{}, ErrNoTwoThirds
	}

	commit := types.Commit{Height: height, Round: round, BlockHash: block.Hash(), Votes: precommits}
	if err := e.Chain.Append(block); err != nil {
		return block, types.Commit{}, err
	}
	return block, commit, nil
}

func (e *Engine) collectVotes(t types.VoteType, height int64, round int, blockHash string) []types.Vote {
	votes := make([]types.Vote, 0)
	for _, v := range e.Validators.Validators {
		if v.Jailed || v.Power <= 0 {
			continue
		}
		votes = append(votes, types.NewVote(t, height, round, v.Address, blockHash))
	}
	return votes
}

func (e *Engine) HasTwoThirds(votes []types.Vote, blockHash string) bool {
	return e.hasTwoThirds(votes, blockHash)
}

func (e *Engine) hasTwoThirds(votes []types.Vote, blockHash string) bool {
	total := e.Validators.TotalPower()
	if total == 0 {
		return false
	}
	seen := map[types.Address]bool{}
	var signed int64
	for _, vote := range votes {
		if !vote.Verify() || vote.BlockHash != blockHash || seen[vote.Validator] {
			continue
		}
		seen[vote.Validator] = true
		signed += e.Validators.PowerOf(vote.Validator)
	}
	return signed*3 > total*2
}

func (e *Engine) DetectAndSlashDoubleSign(votes []types.Vote) []types.Address {
	byKey := map[string]types.Vote{}
	slashed := map[types.Address]bool{}
	for _, vote := range votes {
		key := fmt.Sprintf("%s|%d|%d|%s", vote.Type, vote.Height, vote.Round, vote.Validator)
		old, ok := byKey[key]
		if ok && old.BlockHash != vote.BlockHash {
			e.Validators.Jail(vote.Validator)
			e.Evidence.AddDoubleSign(vote.Validator)
			slashed[vote.Validator] = true
		}
		byKey[key] = vote
	}
	out := make([]types.Address, 0, len(slashed))
	for addr := range slashed {
		out = append(out, addr)
	}
	return out
}
