# Toy Blockchain and Ledger Simulator

A simple blockchain and ledger simulator implemented in **Go**, developed as part of the **Backend Engineering Internship – Golang Developer Assessment**.

The project demonstrates the core concepts behind blockchain technology, including deterministic hashing, Proof-of-Work (PoW) mining, blockchain validation, ledger reconstruction, transaction validation, persistence, and command-line interaction.

Rather than building a production blockchain, the objective is to implement the essential blockchain mechanisms in a clean, modular, and well-tested Go application.

---

# Features

- Deterministic Genesis Block
- Genesis Coinbase (Faucet) Transactions
- SHA-256 Deterministic Block Hashing
- Proof-of-Work (PoW) Mining
- Per-Block Stored Mining Difficulty
- Runtime-Configurable Difficulty, Block Size, and Data File (via flags)
- Transaction Validation
- Ledger Reconstructed Entirely from Blockchain Transactions
- Overspending Protection
- Full Blockchain Validation
- Tamper Detection
- Pending Transaction Pool
- JSON Persistence
- Command-Line Interface (CLI)
- Comprehensive Unit Tests

---

# Project Structure

```
toyblockchain/
│
├── block/
│   ├── block.go
│   └── block_test.go
│
├── chain/
│   ├── blockchain.go
│   ├── blockchain_test.go
│   ├── config.go
│   ├── mining.go
│   ├── mining_test.go
│   ├── pending_storage.go
│   ├── storage.go
│   ├── storage_test.go
│   ├── validation.go
│   └── validation_test.go
│
├── cli/
│   └── cli.go
│
├── ledger/
│   ├── ledger.go
│   ├── transaction.go
│   └── transaction_test.go
│
├── main.go
├── go.mod
├── blockchain.json
├── pending.json
├── README.md
└── report.md
```

---

# Requirements

- Go 1.22 or newer

Check your Go installation:

```bash
go version
```

---

# Building the Project

Clone the repository and build the application.

```bash
go build
```

or run directly:

```bash
go run main.go
```

---

# Configuration Flags

Flags are parsed once, at startup, and must come **before** the command:

```bash
go run main.go -difficulty=N -blocksize=N -data=filename.json <command>
```

| Flag | Description | Default |
|---|---|---|
| `-difficulty` | Leading zeros required when mining | `4` |
| `-blocksize` | Maximum transactions allowed per block | `5` |
| `-data` | Path to the blockchain JSON file | `blockchain.json` |

All flags are optional and fall back to sensible defaults if omitted.

Examples:

```bash
# Mine with a lower difficulty (faster, useful for testing)
go run main.go -difficulty=2 mine

# Limit each block to 2 transactions
go run main.go -blocksize=2 mine

# Use a separate data file, e.g. for a second, independent chain
go run main.go -data=testchain.json print
```

Each block stores the difficulty it was actually mined at, so blocks mined under different `-difficulty` values can coexist correctly in the same chain — validation always checks a block against its own recorded difficulty, not a single global value.

---

# Running the Application

## Show available commands

```bash
go run main.go
```

## Add a transaction

```bash
go run main.go add Alice Bob 20
```

The transaction is validated before being added to the pending transaction pool. Validation includes:

- Amount must be positive
- Sender must have sufficient balance (including any other transactions already pending)

## Mine pending transactions

```bash
go run main.go mine
```

Mining uses the configured difficulty (default, or whatever `-difficulty` was passed). During mining the application displays:

- Difficulty
- Nonce found
- Generated hash
- Mining time

After mining:

- A new block is appended to the blockchain
- The blockchain is saved to disk
- Pending transactions are cleared

## Print the blockchain

```bash
go run main.go print
```

## Display account balances

```bash
go run main.go balance
```

## Validate the blockchain

```bash
go run main.go validate
```

Reports `Valid: true` or `Valid: false`, and on failure names the first block and the specific check that failed.

## Demonstration mode

```bash
go run main.go demo
```

Runs a self-contained walkthrough that:

- Creates sample transactions
- Mines blocks
- Prints the blockchain
- Displays balances
- Validates the honest chain
- Deliberately tampers with a block's transaction
- Validates again, showing detection of the tampering

---

# Running Tests

```bash
go test ./...
```

The test suite covers:

- Deterministic hashing, and that the `Hash` field itself is excluded from its own calculation
- Hash changing when transactions or difficulty change
- Genesis block creation and its coinbase-seeded initial balances
- Mining meeting the configured difficulty target
- Mined nonce reproducing the exact stored hash
- Full-chain validation on an honest chain
- Tamper detection (transaction edits, broken previous-hash links, altered index, altered timestamp, altered proof-of-work)
- Transaction rejection for non-positive amounts and overspending, with balances confirmed unchanged
- Blockchain and pending-pool persistence (save and reload)

---

# Design Decisions

## Block Structure

Each block contains:

- Index
- Timestamp
- Transactions
- Previous Hash
- Nonce
- Difficulty
- Hash

## Genesis Block

The chain starts with a deterministic genesis block. Its previous hash is a fixed value of 64 zero characters, and it is always at index 0.

Unlike later blocks, the genesis block contains two coinbase (faucet) transactions — sender left empty to represent system-issued funds — that introduce the initial currency supply directly on-chain:

```
SYSTEM -> Alice : 100
SYSTEM -> Bob   : 50
```

This means account balances, including the starting ones, are derived entirely by replaying the chain. No balance is stored or set outside of it.

## Deterministic Hashing

Each block's hash is computed with SHA-256 over a stable JSON serialization of the following fields, in this exact order:

1. Index
2. Timestamp
3. Transactions
4. PreviousHash
5. Nonce
6. Difficulty

The `Hash` field itself is intentionally excluded from its own calculation. Hashing the same block twice always produces the same result. Including `Difficulty` in the hash means tampering with a block's recorded difficulty is caught by the same hash-integrity check used for transaction tampering.

## Proof-of-Work Mining

Mining repeatedly increments the nonce until the resulting SHA-256 hash begins with the required number of leading zero hex digits. Each block stores the difficulty it was actually mined at (rather than relying on a single shared value), which is what allows different blocks in the same chain to have been mined at different difficulties and still validate correctly.

Mining reports the difficulty used, the nonce found, the resulting hash, and how long mining took.

## Ledger

The ledger stores no balances of its own between runs. Every balance is reconstructed by replaying every transaction in every block, starting from genesis. A transaction is rejected if its amount is not positive, or if the sender's balance (as of the pending pool, including other not-yet-mined transactions) is insufficient to cover it.

## Blockchain Validation

For every block, validation checks:

- The stored hash matches a fresh recomputation (catches tampering with any field, including transactions and difficulty)
- The previous-hash link matches the prior block's actual hash
- Block indexes are sequential
- Timestamps are non-decreasing
- The block's hash satisfies proof-of-work at its own recorded difficulty

The genesis block is additionally checked for a correct fixed previous-hash and index 0.

Validation returns a clear pass/fail result and, on failure, identifies the first invalid block and which specific check caught the problem.

## Persistence

State is stored as JSON in two files, both configurable via flags:

- `blockchain.json` — the full chain
- `pending.json` — transactions added but not yet mined

Both are loaded automatically on startup (a fresh, empty version is created if the file doesn't exist yet) and saved after relevant commands, so state survives between separate invocations of the program.

## Configurable Parameters (FR-9)

Difficulty, maximum block size, and the data file path are all configurable via command-line flags (`-difficulty`, `-blocksize`, `-data`), parsed once at startup in `main.go` and bound directly to package-level variables in `chain/config.go`. All three have sensible defaults and can be left unset.

---

# Known Limitations

This project is intended for educational purposes and intentionally omits several features found in production blockchains:

- Peer-to-peer networking
- Distributed consensus
- Public/private key cryptography or digital signatures
- Wallet management
- Merkle trees
- Smart contracts
- Automatic difficulty adjustment (retargeting)
- Mining rewards beyond the genesis coinbase transactions
- Chain forks or fork resolution

Additionally, worth being explicit about:

- **Breaking changes to the `Block` struct affect old data files.** Adding the `Difficulty` field to `Block` changed what goes into the hash calculation. A `blockchain.json` saved by an earlier version of this program (before `Difficulty` existed) will fail validation if loaded by the current version, since its originally stored hash was computed without that field. There is no schema versioning to handle this gracefully — old data files should be deleted rather than reused across versions of the code.
- **Pending-pool balance checks are advisory, not final.** A transaction is validated against a temporary ledger clone at `add` time (accounting for other pending transactions), but the ledger itself isn't finalized until the block is actually mined.

---

# Technologies Used

- Go 1.22+
- Standard library only: `crypto/sha256`, `encoding/json`, `encoding/hex`, `flag`, `fmt`, `os`, `strconv`, `strings`, `testing`, `time`

No third-party libraries were used.

---

# Assessment Notes

This project was implemented to satisfy the Backend Engineering Internship Golang Developer Assessment. Key implementation decisions:

- Deterministic genesis block, seeded with on-chain coinbase transactions rather than hardcoded ledger credits
- Ledger reconstructed entirely from blockchain history — no balances stored outside the chain
- SHA-256 deterministic hashing, with difficulty included in the hashed fields
- Proof-of-Work mining with difficulty stored per block, so validation always checks a block against the difficulty it was actually mined at
- Full blockchain and pending-pool persistence using JSON
- Difficulty, block size, and data file path configurable via CLI flags with sensible defaults (FR-9)
- Command-line interface covering all required operations
- Unit tests covering hashing determinism, mining, validation (including multiple distinct tamper scenarios), transaction rejection, and persistence

See `report.md` for the research component: tamper-evidence experiment output, a difficulty-versus-mining-time table and analysis, the hashing/validation design write-up, and answers to the discussion questions.

---

# Author

Developed as part of the **Backend Engineering Internship – Golang Developer Assessment** using the Go programming language.
