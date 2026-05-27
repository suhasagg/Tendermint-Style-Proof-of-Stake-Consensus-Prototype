# Tendermint-Style Proof-of-Stake Consensus Prototype in Go

This repository is prototype of a Tendermint-style Proof-of-Stake consensus system implemented from scratch in Go.

It models validator power, proposer selection, block proposal, prevote, precommit, commit, weighted `+2/3` voting power, basic evidence detection, and slashing/jaailing for double-signing.

## What This Prototype Implements

### Core Concepts

- Validator set with voting power
- Round-robin proposer selection
- In-memory mempool
- Block creation from pending transactions
- Deterministic block hashing
- Prevote phase
- Precommit phase
- Commit when more than two-thirds of active voting power signs
- Basic vote signature simulation using SHA-256
- Duplicate vote protection
- Double-sign evidence detection
- Slashing by jailing a validator
- Unit tests for consensus success, weighted voting, double-signing, and invalid signatures
- End-to-end tests for multi-height commits, mempool draining, slashing impact, and no-validator failure

## Repository Structure

```text
tendermint-pos-prototype/
├── cmd/
│   └── demo/
│       └── main.go              # Demo executable
├── consensus/
│   ├── engine.go                # Consensus engine, mempool, chain, evidence pool
│   ├── engine_test.go           # Unit tests
│   └── e2e_test.go              # End-to-end consensus tests
├── types/
│   └── types.go                 # Validator, block, vote, commit types
├── go.mod
└── README.md
```

## Architecture

```text
                +-------------------+
                |     Mempool       |
                | pending txs       |
                +---------+---------+
                          |
                          v
+-------------+    +------+-------+      +----------------+
| Validator   | -> |  Proposer    | ---> | Proposed Block |
| Set + Power |    | Selection    |      +-------+--------+
+------+------+    +--------------+              |
       |                                          v
       |                                  +-------+--------+
       |                                  |    Prevotes    |
       |                                  +-------+--------+
       |                                          |
       |                              +2/3 voting power?
       |                                          |
       |                                          v
       |                                  +-------+--------+
       |                                  |   Precommits   |
       |                                  +-------+--------+
       |                                          |
       |                              +2/3 voting power?
       |                                          |
       v                                          v
+------+----------------+                 +-------+--------+
| Evidence + Slashing   |                 | Commit Block   |
+-----------------------+                 +----------------+
```

## Simplified Tendermint Flow

For each height:

1. Select proposer from active validators.
2. Proposer creates a block using transactions from the mempool.
3. Validators prevote for the proposed block hash.
4. If prevotes reach more than two-thirds of active voting power, validators precommit.
5. If precommits reach more than two-thirds of active voting power, the block is committed.
6. The block is appended to the local chain.

```text
Height H, Round R

Propose Block B
      |
      v
Prevote B --------------> need > 2/3 voting power
      |
      v
Precommit B ------------> need > 2/3 voting power
      |
      v
Commit B
```

## How Voting Power Works

Validators have stake-weighted voting power:

```go
[]types.Validator{
    {Address: "val1", Power: 40},
    {Address: "val2", Power: 30},
    {Address: "val3", Power: 20},
    {Address: "val4", Power: 10},
}
```

Total active power is `100`.

A block commits only if signed power is greater than `2/3`:

```text
val1 + val2 = 40 + 30 = 70  => commit succeeds
val1 + val3 = 40 + 20 = 60  => commit fails
```

The code checks this with:

```go
return signed*3 > total*2
```

## Running the Demo

```bash
go run ./cmd/demo
```

Expected output:

```text
Committed block
height: 1
proposer: val2
hash: <block-hash>
commit votes: 4
```

The proposer is selected using a simple round-robin rule:

```go
idx := int((height + int64(round)) % int64(len(active)))
```

## Running Tests

Run these commands from the repository root:

```bash
cd tendermint-pos-prototype
```

### 1. Run all tests

This runs every unit test and E2E test in all Go packages:

```bash
go test ./...
```

Expected result:

```text
ok  	tendermint-pos-prototype/consensus
?   	tendermint-pos-prototype/cmd/demo [no test files]
?   	tendermint-pos-prototype/types [no test files]
```

### 2. Run tests with verbose output

Use this when you want to see each test name:

```bash
go test ./... -v
```

### 3. Run only unit tests

Unit tests live in `consensus/engine_test.go`:

```bash
go test ./consensus -v -run 'TestRunHeightCommitsBlock|TestTwoThirdsWeightedVoting|TestDoubleSignSlashesValidator|TestInvalidSignatureIgnored'
```

### 4. Run only E2E tests

E2E tests live in `consensus/e2e_test.go` and use test names containing `E2E`:

```bash
go test ./consensus -v -run E2E
```

Expected output includes:

```text
=== RUN   TestE2EConsensusCommitsMultipleBlocks
--- PASS: TestE2EConsensusCommitsMultipleBlocks
PASS
```

### 5. Common mistake

Use:

```bash
go test ./...
```

Do **not** use:

```bash
go test ./..
```

`./...` means this module and all subpackages.  
`./..` means the parent directory, for example `~/Downloads`, which usually has no Go files.

### Tests Included

| Test file | Type | What it checks |
|---|---|---|
| `consensus/engine_test.go` | Unit tests | Block commit, weighted voting, double-sign slashing, invalid vote rejection |
| `consensus/e2e_test.go` | E2E tests | Multi-block chain progress, mempool draining, previous-hash linking, slashing impact, no-active-validator failure |


## Important Files

### `types/types.go`

Defines:

- `Validator`
- `ValidatorSet`
- `Block`
- `Vote`
- `Commit`

The `Vote` type includes a simulated signature:

```go
func NewVote(t VoteType, height int64, round int, validator Address, blockHash string) Vote
```

In production, this would use Ed25519/secp256k1 signatures over canonical vote bytes.

### `consensus/engine.go`

Defines:

- `Engine`
- `Mempool`
- `Blockchain`
- `EvidencePool`

Main consensus entry point:

```go
func (e *Engine) RunHeight(round int) (types.Block, types.Commit, error)
```

This executes:

1. proposer selection
2. block creation
3. prevote collection
4. precommit collection
5. commit creation
6. chain append

## Unit Tests Included

### `TestRunHeightCommitsBlock`

Checks that a block is proposed, receives votes, commits, and is appended to the chain.

### `TestTwoThirdsWeightedVoting`

Checks weighted voting power rules.

### `TestDoubleSignSlashesValidator`

Creates two conflicting votes from the same validator at the same height/round/type and verifies that the validator is jailed.

### `TestInvalidSignatureIgnored`

Ensures tampered votes do not count toward consensus.

## What Is Simplified Compared to Real Tendermint

| Area | Prototype | Real Tendermint / CometBFT |
|---|---|---|
| Networking | none | peer-to-peer gossip |
| Cryptography | SHA-256 simulated signatures | real validator private keys |
| Consensus locking | not implemented | locked rounds, valid rounds, POL rules |
| Timeouts | not implemented | propose/prevote/precommit/commit timeouts |
| Storage | in-memory | persistent block store/state store/WAL |
| ABCI | not implemented | application interface for state transitions |
| Fork accountability | basic double-sign only | full evidence handling |
| Validator changes | static set | app-driven validator updates |
| Mempool | simple FIFO | gossip, recheck, priority, eviction |

## How To Extend This Prototype

Good next improvements:

1. Add real Ed25519 validator keys.
2. Add a P2P network layer using libp2p or Go channels for simulation.
3. Add consensus timeouts and multiple rounds.
4. Add lock/unlock rules for prevote and precommit.
5. Add ABCI-style application execution.
6. Add validator set updates at end block.
7. Add persistent block/state storage.
8. Add light-client verification with validator signatures.
9. Add slashing for downtime.
10. Add a CLI to start multiple validator nodes.
