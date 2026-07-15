# Toy Blockchain and Ledger Simulator

A simple blockchain and ledger simulator implemented in **Go**, developed as part of the **Backend Engineering Internship – Golang Developer Assessment**.

The project demonstrates the core concepts behind blockchain technology, including deterministic hashing, Proof-of-Work (PoW) mining, blockchain validation, ledger reconstruction, transaction validation, persistence, and command-line interaction. It also implements every optional stretch goal from the assessment brief: Ed25519 digital signatures, a Merkle root per block, concurrent mining, automatic difficulty retargeting, and longest-valid-chain fork resolution.

Rather than building a production blockchain, the objective is to implement the essential blockchain mechanisms in a clean, modular, and well-tested Go application.

---

# Features

**Core**
- Deterministic Genesis Block
- Genesis Coinbase (Faucet) Transactions
- SHA-256 Deterministic Block Hashing
- Proof-of-Work (PoW) Mining
- Per-Block Stored Mining Difficulty, With a Configured Minimum
- Runtime-Configurable Difficulty, Block Size, and Data File (via flags)
- Transaction Validation, Including Rejection of Empty/Reserved Senders at the CLI
- Ledger Reconstructed Entirely from Blockchain Transactions
- Overspending Protection, Including Full Ledger-Replay Validation Across the Chain
- Full Blockchain Validation, Including Validation on Load From Disk
- Tamper Detection
- Pending Transaction Pool, Re-Validated at Mine Time
- JSON Persistence
- Command-Line Interface (CLI)
- Comprehensive Unit Tests

**Stretch goals**
- **Digital signatures** — every non-coinbase transaction is signed with Ed25519 and verified during ledger replay, with first-seen public-key binding per sender name to prevent impersonation
- **Wallet management** — named accounts are backed by real Ed25519 key pairs, persisted to `wallet.json` and reused across runs
- **Merkle root** — each block summarises its transactions with a Merkle root instead of hashing the raw transaction list directly, and the root itself feeds into the block hash
- **Concurrent mining** — the nonce search is parallelised across goroutines (`runtime.NumCPU()` workers by default) and stops cleanly as soon as any worker finds a valid nonce
- **Difficulty retargeting** — mining difficulty adjusts automatically every few blocks to keep block time roughly constant, with a configured floor
- **Fork resolution** — a competing chain can be loaded from a file and is adopted if it's longer (or, at equal length, has done more cumulative work), following a longest-valid-chain rule

---

# Project Structure
toyblockchain/
│
├── block/
│   ├── block.go
│   ├── block_test.go
│   ├── merkle.go
│   └── merkle_test.go
│
├── chain/
│   ├── blockchain.go
│   ├── blockchain_test.go
│   ├── config.go
│   ├── fork.go
│   ├── fork_test.go
│   ├── mining.go
│   ├── mining_test.go
│   ├── mining_concurrent.go
│   ├── mining_concurrent_test.go
│   ├── pending_storage.go
│   ├── retarget.go
│   ├── retarget_test.go
│   ├── storage.go
│   ├── storage_test.go
│   ├── test_helpers.go
│   ├── validation.go
│   └── validation_test.go
│
├── cli/
│   ├── cli.go
│   └── cli_test.go
│
├── crypto/
│   ├── keys.go
│   ├── signature.go
│   └── signature_test.go
│
├── ledger/
│   ├── ledger.go
│   ├── ledger_test.go
│   ├── signature.go
│   ├── transaction.go
│   ├── transaction_test.go
│   └── verify.go
│
├── wallet/
│   ├── wallet.go
│   └── wallet_test.go
│
├── main.go
├── go.mod
├── README.md
└── report.md

`blockchain.json`, `pending.json`, and `wallet.json` are created automatically the first time the program runs and are not committed to version control — see `.gitignore`. `wallet.json` in particular contains plaintext private keys and must never be committed (see the Wallet section below).

---

# Requirements

- Go 1.22 or newer (per `go.mod`)

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
| `-difficulty` | Mining difficulty (leading zeros), honored until retargeting kicks in | `4` |
| `-blocksize` | Maximum transactions allowed per block | `5` |
| `-data` | Path to the blockchain JSON file | `blockchain.json` |

All flags are optional and fall back to sensible defaults if omitted. If `-difficulty` is set below the configured minimum (`MinDifficulty`), it is raised to the minimum automatically, with a message explaining why.

Once a chain has enough history (more than `RetargetInterval` blocks), difficulty adjusts automatically — see **Difficulty Retargeting** below — and `-difficulty` is no longer honored directly; it only matters for the first few blocks of a fresh chain.

Examples:

```bash
# Mine with a lower difficulty (faster, useful for testing)
go run main.go -difficulty=2 mine

# Limit each block to 2 transactions
go run main.go -blocksize=2 mine

# Use a separate data file, e.g. for a second, independent chain
go run main.go -data=testchain.json print
```

Each block stores the difficulty it was actually mined at, so blocks mined under different difficulty settings can coexist correctly in the same chain — validation always checks a block against its own recorded difficulty, not a single global value, and rejects any block whose recorded difficulty falls below the configured minimum.

---

# Running the Application

## Show available commands

```bash
go run main.go
```

## Wallets and signing

There is no separate `keygen` command. Instead, the first time a given account name is used in `add` or `demo`, a fresh Ed25519 key pair is generated for it automatically via `wallet.GetOrCreate`, and persisted to `wallet.json`. Every later transaction from that name reuses the same stored key pair. This keeps the CLI's existing "sender by name" interface (`add Alice Bob 20`) while every transaction underneath is genuinely signed and verified — a name can't be impersonated by a second, different key pair once it has transacted at least once (see **Digital Signatures** below).

## Add a transaction

```bash
go run main.go add Alice Bob 20
```

The sender's key pair is fetched (or created, on first use) from the wallet, the transaction is signed, and then validated before being added to the pending transaction pool. Validation includes:

- Amount must be positive
- Sender must not be empty or the reserved name `SYSTEM` (case-insensitive) — this prevents user input from minting funds out of nowhere
- Sender must have sufficient balance (including any other transactions already pending)

## Mine pending transactions

```bash
go run main.go mine
```

Before mining, the pending pool (loaded from `pending.json`, which is user-editable) is re-validated: sender validity, **signature validity**, and balances are re-checked against a temporary ledger, and mining aborts if anything invalid is found — this is also what catches a hand-edited pending pool, since an edited transaction's signature no longer matches (see `TestMineRejectsHandEditedOverspend`). Mining uses the difficulty returned by `NextDifficultyFor` (the configured `-difficulty` early in a chain's life, then the automatically retargeted value once enough history exists), and runs across multiple goroutines. During mining the application displays:

- Difficulty
- Number of concurrent workers
- Merkle root
- Nonce found
- Generated hash
- Mining time

After mining:

- A new block is appended to the blockchain
- The blockchain is saved to disk explicitly (mining itself has no persistence side effect)
- Only as many transactions as `-blocksize` allows are mined; any remainder stays in the pending pool for the next `mine` call, rather than blocking mining entirely

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

Reports `Valid: true` or `Valid: false`, and on failure names the first block and the specific check that failed (Merkle root mismatch, hash mismatch, broken previous-hash link, bad index, bad timestamp, invalid proof-of-work, or a ledger-replay failure such as an invalid signature or overspend). The blockchain is also validated automatically whenever it is loaded from disk at startup, so a hand-edited `blockchain.json` is caught immediately rather than only when `validate` happens to be run.

## Resolve a fork

```bash
go run main.go resolve <candidate_chain.json>
```

Loads a competing chain from a file and applies the longest-valid-chain rule: the candidate is first fully validated on its own, then compared against the current chain. It's adopted if it has more blocks, or — at equal length — more cumulative difficulty ("work"). If accepted, the current chain is replaced and saved to disk; otherwise nothing changes. See **Fork Resolution** below for the exact rule and why genesis is checked first.

## Demonstration mode

```bash
go run main.go demo
```

Runs a self-contained walkthrough that:

- Creates two signed sample transactions (`Alice -> Bob : 20`, `Bob -> Charlie : 10`), generating wallet key pairs for Alice and Bob if they don't already exist
- Mines each into its own block
- Saves the resulting chain to disk
- Prints the blockchain
- Validates the chain and reports the result

`demo` does not currently tamper with a block itself — for a tamper-and-detect walkthrough, see the tamper-evidence experiment in `report.md`, which shows the before/after validation output from editing a transaction directly in `blockchain.json`.

---

# Running Tests

```bash
go test ./...
```

The test suite covers:

- Deterministic hashing, and that the `Hash` field itself is excluded from its own calculation
- Hash changing when transactions or difficulty change
- Merkle root determinism, sensitivity to transaction changes, odd-count handling, and that empty/nil transaction lists produce the same root
- Genesis block creation, its coinbase-seeded initial balances, and that genesis is fully deterministic (fixed timestamp and hash across independently created chains)
- Mining meeting the configured difficulty target, both sequentially and concurrently, and that a single-worker concurrent run matches sequential mining
- Mined nonce reproducing the exact stored hash, in both mining modes
- Automatic difficulty retargeting: increasing when blocks are mined fast, decreasing when slow, never dropping below the configured minimum, not retargeting before enough history exists, and correctly handing off between the requested difficulty and the retargeted one
- Full-chain validation on an honest chain
- Six distinct tamper scenarios, each verified to fail at the specific check it targets: a raw transaction edit (hash mismatch), a direct Merkle root edit, a broken previous-hash link, an altered index, an altered timestamp, and a block whose hash doesn't satisfy its recorded proof-of-work
- A chain containing an overspending transaction (smuggled in via a hand-edited pending pool) failing validation through ledger replay, not just being silently skipped
- Transaction rejection for non-positive amounts and overspending at the ledger level, with balances confirmed unchanged in both cases
- Correct minting behaviour for empty-sender (coinbase) transactions at the ledger level, and rejection of empty/reserved senders at the CLI level
- Ed25519 key pair generation, sign/verify round trips, rejection of tampered messages and wrong public keys, and graceful (non-panicking) handling of malformed hex input
- Wallet `GetOrCreate` returning a stable key pair for a repeated name, wallet save/load round trips, auto-creation of a missing wallet file, and reverse name lookup by public key
- Fork resolution: accepting a genuinely longer valid chain, rejecting shorter or equal-length-and-equal-work chains, and rejecting a candidate that fails its own validation
- Blockchain and pending-pool persistence (save and reload), including that a reloaded pending transaction's signature survives the round trip

---

# Design Decisions

## Block Structure

Each block contains:

- Index
- Timestamp
- Transactions
- Merkle Root
- Previous Hash
- Nonce
- Difficulty
- Hash

## Genesis Block

The chain starts with a deterministic genesis block, built entirely from fixed constants (`GenesisTimestamp`, `GenesisPreviousHash`) rather than the current time — two independently created blockchains produce byte-identical genesis blocks, including the same hash. `GenesisPreviousHash` is generated with `strings.Repeat("0", 64)` rather than a hand-typed literal, to guarantee it's exactly 64 characters (matching the shape of a real SHA-256 hex digest) without relying on manually counting zeros. The genesis block is always at index 0.

Unlike later blocks, the genesis block contains two coinbase (faucet) transactions — sender left empty to represent system-issued funds — that introduce the initial currency supply directly on-chain:
SYSTEM -> Alice : 100
SYSTEM -> Bob   : 50

This means account balances, including the starting ones, are derived entirely by replaying the chain. No balance is stored or set outside of it.

## Deterministic Hashing

Each block's hash is computed with SHA-256 over a stable JSON serialization of the following fields, in this exact order:

1. Index
2. Timestamp
3. MerkleRoot
4. PreviousHash
5. Nonce
6. Difficulty

The `Hash` field itself is intentionally excluded from its own calculation. Hashing the same block twice always produces the same result. The raw `Transactions` slice is not hashed directly — instead its Merkle root is computed first and that root feeds into the block hash (see below). Including `Difficulty` in the hash means tampering with a block's recorded difficulty is caught by the same hash-integrity check used for transaction tampering.

## Merkle Root

Each block's transaction list is summarised into a single Merkle root: every transaction is individually hashed with SHA-256, then hashes are paired and combined (an odd one out is paired with itself) repeatedly up the tree until one root hash remains. An empty transaction list produces the SHA-256 hash of an empty byte slice, deterministically, so genesis-style blocks with no transactions still have a well-defined root.

This root — not the raw transaction list — is one of the fields that feeds into the block's own hash. Practically, this means changing any single transaction changes the Merkle root, which changes the block hash, which is exactly what `ValidateChain` catches as a "Merkle root mismatch" before it even gets to the hash-mismatch check. It also means a block's transaction integrity can be verified without necessarily re-hashing every transaction from scratch every time, which is closer to how a real chain uses Merkle trees for efficient partial verification — though this toy always recomputes the full root rather than proving individual-transaction inclusion.

## Digital Signatures

Every account is backed by an Ed25519 key pair (`crypto/ed25519`, standard library — no external dependency). A `Transaction` carries a `PublicKey` and a `Signature`, produced by signing a stable byte encoding of `Sender:Receiver:Amount`. `ledger.ApplyTransaction` verifies this signature for every non-coinbase transaction before applying any balance change, and additionally enforces that each sender name is permanently bound to the first public key ever seen signing on its behalf — a second key pair cannot later "become" the same named account. This closes a gap that existed before signatures: without them, nothing stopped one user from constructing a transaction claiming to be from someone else's account.

Because this check lives inside `ApplyTransaction`, it's automatically exercised everywhere a transaction is applied: adding to the pending pool, mining, and full-chain validation (including on load from disk) — there's no separate signature-checking code path to keep in sync.

## Wallet

The `wallet` package maps human-readable account names to Ed25519 key pairs, persisted as JSON in `wallet.json`. `GetOrCreate(name)` returns the existing key pair for a known name, or generates and persists a new one the first time a name is seen — this is what lets the CLI keep accepting plain names like `Alice` while every transaction underneath is properly signed. Saves are atomic (temp file + rename), matching the pattern already used for blockchain and pending-pool persistence.

**Security note:** this stores private keys in plaintext JSON on disk, which is appropriate for a toy/educational project but not how a production system would manage keys — a real wallet would use an OS keychain, hardware security module, or at minimum encrypt the file at rest. `wallet.json` is excluded via `.gitignore` and must never be committed.

## Proof-of-Work Mining

Mining repeatedly increments the nonce until the resulting SHA-256 hash begins with the required number of leading zero hex digits. Each block stores the difficulty it was actually mined at (rather than relying on a single shared value), which is what allows different blocks in the same chain to have been mined at different difficulties and still validate correctly. Validation additionally enforces a configured minimum difficulty (`MinDifficulty`), so a block can't claim to have been mined at a suspiciously low difficulty to make forging the rest of the chain cheaper.

Mining reports the difficulty used, the number of workers, the Merkle root, the nonce found, the resulting hash, and how long mining took.

## Concurrent Mining

`MineBlockConcurrent` splits the nonce search across `DefaultMiningWorkers` goroutines (`runtime.NumCPU()` by default). Each worker searches a disjoint slice of the nonce space (starting at its worker ID, stepping by the worker count) using its own local copy of the block, so no worker mutates shared state while searching. An `atomic.Bool` flag and `CompareAndSwap` ensure exactly one worker's result is accepted as the winner even if multiple workers happen to find a valid nonce around the same time, and all workers stop as soon as the flag is set. This is a natural fit for Go's goroutines and channels-free `sync`/`sync/atomic` primitives, and a single-worker run is tested to still behave correctly (functionally equivalent to sequential mining, just through the concurrent code path).

## Difficulty Retargeting

Every `RetargetInterval` blocks, `CalculateNextDifficulty` compares the actual time taken to mine the last interval's worth of blocks against the target (`RetargetInterval * TargetBlockTimeSeconds`). If blocks were mined faster than target, difficulty increases by `MaxDifficultyStep`; if slower, it decreases by the same step — always floored at `MinDifficulty`. Before enough history exists to measure a full interval, `NextDifficultyFor` falls back to honoring the difficulty requested via configuration, which is what allows `-difficulty` to control the very first blocks of a fresh chain before retargeting has enough data to take over.

## Fork Resolution

`ResolveFork` accepts a candidate chain and applies a longest-valid-chain rule with a work-based tiebreaker: the candidate must first pass its own full `ValidateChain` check and share the same genesis block hash as the current chain (rejecting an unrelated chain outright). It's then adopted if it has strictly more blocks than the current chain, or — at equal length — more cumulative "work" (the sum of each block's recorded difficulty). A shorter or equal-weaker candidate is rejected and the current chain is left untouched. This mirrors the real-world "longest valid chain wins" consensus rule at a conceptual level, without any actual peer-to-peer network to source competing chains from — candidates are supplied manually via the `resolve <file>` command.

## Ledger

The ledger stores no balances of its own between runs. Every balance is reconstructed by replaying every transaction in every block, starting from genesis. A transaction is rejected if its amount is not positive, if its signature doesn't verify (or was signed by a different key than the one already registered for that sender name), or if the sender's balance (as of the pending pool, including other not-yet-mined transactions) is insufficient to cover it. An empty sender is treated as a coinbase/faucet mint and is never debited or signature-checked — but the CLI's `add` command rejects an empty or reserved (`SYSTEM`) sender coming from user input, so this minting path is only ever reachable from genesis, not from user commands.

## Blockchain Validation

For every block, validation checks, in order:

- The Merkle root matches a fresh recomputation from the block's transactions
- The stored hash matches a fresh recomputation (catches tampering with any field, including the Merkle root and difficulty)
- The previous-hash link matches the prior block's actual hash
- Block indexes are sequential
- Timestamps are non-decreasing
- The block's recorded difficulty is at or above the configured minimum
- The block's hash satisfies proof-of-work at its own recorded difficulty
- Every transaction in the block replays cleanly against a running ledger — signature validity, sender identity binding, a non-positive amount, or an overspend anywhere in the chain's history all fail validation, rather than being silently skipped

The genesis block is additionally checked for a correct fixed previous-hash and index 0.

Validation returns a clear pass/fail result and, on failure, identifies the first invalid block and which specific check caught the problem. The blockchain is also run through this same validation automatically when loaded from disk at startup, and a candidate chain is run through it as the first step of fork resolution.

## Persistence

State is stored as JSON in three files, two of them configurable via flags:

- `blockchain.json` — the full chain
- `pending.json` — transactions added but not yet mined
- `wallet.json` — named accounts and their Ed25519 key pairs

All are loaded automatically on startup (a fresh, empty version is created if the file doesn't exist yet) and saved after relevant commands, so state survives between separate invocations of the program. All three use an atomic temp-file-and-rename write pattern, so a crash mid-write can't corrupt an existing file. Chain persistence is an explicit step after mining rather than a side effect inside `AddBlock`, so core chain logic stays decoupled from disk I/O.

## Configurable Parameters (FR-9)

Difficulty, maximum block size, and the data file path are all configurable via command-line flags (`-difficulty`, `-blocksize`, `-data`), parsed once at startup in `main.go` and bound directly to package-level variables in `chain/config.go`. All three have sensible defaults and can be left unset. A configured difficulty below `MinDifficulty` is automatically raised, rather than silently producing blocks that later fail validation. Note that `-difficulty` is only honored until enough chain history exists for automatic retargeting to take over (see **Difficulty Retargeting**).

---

# Known Limitations

This project is intended for educational purposes and intentionally omits several features found in production blockchains:

- Peer-to-peer networking
- Distributed consensus between independent nodes
- Smart contracts, a virtual machine, or EVM/Solidity-style execution
- Mining rewards beyond the genesis coinbase transactions
- Merkle *proofs* of individual-transaction inclusion — the Merkle root is computed and verified as a whole, but there's no API to prove a single transaction belongs to a block without recomputing the full root

Additionally, worth being explicit about:

- **Breaking changes to the `Block` struct affect old data files.** Adding fields like `MerkleRoot` and `Difficulty` changed what goes into the hash calculation. A `blockchain.json` saved by an earlier version of this program will fail validation if loaded by the current version, since its originally stored hash was computed without those fields. There is no schema versioning to handle this gracefully — old data files should be deleted rather than reused across versions of the code.
- **`wallet.json` stores private keys in plaintext.** This is acceptable for a toy/educational project but is not how a production system would manage keys — see the Wallet design section above. This file must never be committed to version control (it's excluded via `.gitignore`).
- **Pending-pool balance checks are advisory, not final.** A transaction is validated against a temporary ledger clone at `add` time (accounting for other pending transactions), but the ledger itself isn't finalized until the block is actually mined. `mine` re-validates the pending pool (including signatures) before mining, but a transaction can still be invalidated between `add` and `mine` if other transactions are added or the pool is hand-edited in between.
- **`demo` ignores errors from `AddBlock`.** If mining fails inside `demo` (for example, an oversized batch), the error is currently not checked, so failure would go unreported. This doesn't affect correctness of the main CLI paths (`add`/`mine`), which do check `AddBlock`'s return value.
- **Fork resolution has no automatic source of candidates.** `resolve` accepts a chain from a file the user provides — there's no peer-to-peer layer to discover or fetch competing chains from, consistent with networking being explicitly out of scope for this assessment.

---

# Technologies Used

- Go 1.22+
- Standard library only: `crypto/sha256`, `crypto/ed25519`, `crypto/rand`, `encoding/json`, `encoding/hex`, `flag`, `fmt`, `os`, `path/filepath`, `runtime`, `strconv`, `strings`, `sync`, `sync/atomic`, `testing`, `time`

No third-party libraries were used.

---

# Assessment Notes

This project was implemented to satisfy the Backend Engineering Internship Golang Developer Assessment. Key implementation decisions:

- Deterministic genesis block, built from fixed constants (including a programmatically generated 64-character previous-hash) and seeded with on-chain coinbase transactions rather than hardcoded ledger credits
- Ledger reconstructed entirely from blockchain history — no balances stored outside the chain
- SHA-256 deterministic hashing over a Merkle root plus block metadata, rather than hashing the raw transaction list directly
- Proof-of-Work mining, parallelised across goroutines, with difficulty stored per block, floored at a configured minimum, and automatically retargeted every few blocks to keep block time roughly constant
- Ed25519 transaction signatures with first-seen public-key binding per sender name, verified as part of every ledger apply — pending pool, mining, and full validation all share this one code path
- Named accounts backed by a persisted wallet of real key pairs, so the CLI keeps its simple "sender by name" interface without sacrificing genuine signing
- Longest-valid-chain fork resolution with a cumulative-work tiebreak, exposed via a `resolve <file>` command
- Chain validation includes full ledger replay, and runs automatically on load from disk and at the start of fork resolution, not just when `validate` is invoked
- Full blockchain, pending-pool, and wallet persistence using JSON with atomic writes
- Difficulty, block size, and data file path configurable via CLI flags with sensible defaults (FR-9)
- Command-line interface covering all required operations plus fork resolution, with CLI-level rejection of empty/reserved senders
- Unit tests covering hashing determinism, Merkle root behaviour, mining (sequential and concurrent), retargeting, validation (six distinct, individually-verified tamper scenarios plus ledger-replay failure), transaction rejection, signature verification, wallet persistence, fork resolution, and general persistence

See `report.md` for the research component: tamper-evidence experiment output, a difficulty-versus-mining-time table and analysis, the hashing/validation design write-up, and answers to the discussion questions.

---

# Author

Developed as part of the **Backend Engineering Internship – Golang Developer Assessment** using the Go programming language