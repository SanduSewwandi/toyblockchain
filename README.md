# Networked Multi-Node Blockchain — Round 2

A small peer-to-peer network of blockchain nodes, extending the Round 1 toy
blockchain with gossip, sync, and fork resolution. Everything runs locally
as separate processes on different ports.

## Table of contents

- [What changed from Round 1](#what-changed-from-round-1)
- [Requirements](#requirements)
- [Project layout](#project-layout)
- [Running a single node](#running-a-single-node)
- [Starting a local cluster](#starting-a-local-cluster-3-nodes)
- [Developing in VS Code](#developing-in-vs-code)
- [HTTP API](#http-api)
- [Design decisions](#design-decisions)
- [Known limitations](#known-limitations)
- [Tests](#tests)
- [Acceptance criteria](#acceptance-criteria)
- [Research report](#research-report)

## What changed from Round 1

- Refactored the single-process chain into reusable packages: `block`,
  `chain`, `ledger`, `crypto`, `wallet`, `node`, `cli`.
- Added ed25519 key pairs (`crypto` package) — transactions are now signed
  and verified everywhere they enter a node, instead of trusting a
  free-form sender string.
- Added a `node` package: each node wraps a `chain.Blockchain` with a
  pending pool, peer set, and an HTTP API (`node/server.go`,
  `node/handlers.go`).
- Added transaction and block gossip with de-duplication (`seenTx`,
  `seenBlocks` maps keyed by signature/hash).
- Added chain sync (`node/sync.go`) — incremental block-by-block
  catch-up via `GET /height` and `GET /blocks/{index}`, falling back to
  a full `/chain` compare only when a fork is detected — and fork
  resolution (`chain/fork.go`, `chain.ResolveFork`) so independent nodes
  converge on one history.
- Added peer health checks and peer discovery (`node/peers.go`).
- Added node identity and chain persistence across restarts
  (`node/identity.go`, `node/persistence.go`).
- Concurrency: all shared node state (`Blockchain`, `Pending`, `Peers`) is
  guarded by a single `sync.RWMutex` in `Node`, verified race-free with
  `go test -race ./...` and a dedicated stress test
  (`TestConcurrentMiningGossipAndReadsAreRaceFree`).
- Kept the Round 1 hashing, Merkle root, proof-of-work, difficulty
  retargeting, and ledger validation logic unchanged — the network layer
  is built on top of it, not a rewrite of it.

## Requirements

- Go 1.22 or newer
- No external services or third-party blockchain libraries required

## Project layout

```
block/     block struct, deterministic hashing, Merkle root
chain/     blockchain, mining, difficulty retargeting, validation, fork resolution
ledger/    account balances, transaction application
crypto/    ed25519 key generation, signing, verification
wallet/    named local accounts mapped to key pairs (CLI convenience)
node/      networked node: HTTP API, gossip, sync, fork handling, peer health
cli/       round 1 single-process CLI (kept for reference and local testing)
cmd/       entry point(s) — cmd/node starts a networked node process
scripts/   cluster launcher script(s)
logs/      cluster run logs (if produced by the launcher script)
```

## Running a single node

```bash
go run ./cmd/node -addr=:8001 -peers=""
```

Flags (see `cmd/node/main.go`):

| Flag | Default | Description |
|---|---|---|
| `-addr` | `:8001` | address this node listens on |
| `-peers` | `""` | comma-separated list of peer addresses |
| `-mine-interval` | `5s` | how often the node attempts to mine pending transactions |
| `-no-mine` | `false` | disable this node's mining loop (still validates and gossips) |
| `-peer-health-interval` | `10s` | how often peer health is checked and peer lists exchanged |
| `-data` | `node_data/chain_<addr>.json` | blockchain persistence file |
| `-keyfile` | `node_data/identity_<addr>.json` | node identity key file |
| `-persist-interval` | `10s` | how often the chain is saved to disk |

On startup, a node loads (or creates) its identity key pair and its
persisted chain, starts its HTTP server, attempts an initial sync from any
configured peers, then begins its mining and peer-health loops.

## Starting a local cluster (3+ nodes)

Start the three-node local cluster with:

```bash
./scripts/cluster.sh start
```

Check the cluster status with:

```bash
./scripts/cluster.sh status
```

Stop the cluster with:

```bash
./scripts/cluster.sh stop
```

Or manually, in separate terminals:

```bash
go run ./cmd/node -addr=:8001 -peers=:8002,:8003
go run ./cmd/node -addr=:8002 -peers=:8001,:8003
go run ./cmd/node -addr=:8003 -peers=:8001,:8002
```

Each node persists its chain and identity under `node_data/`, and logs
peer connections, gossip, block acceptance, and reorganisations to stdout.

To stop the cluster, send `SIGINT`/`SIGTERM` (e.g. `Ctrl+C` or
`kill <pid>`) to each process — each node saves its chain to disk and
shuts down its HTTP server cleanly before exiting.

## Developing in VS Code

This project was built and tested in VS Code with the official Go
extension.

**Setup**

1. Install the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go) (`golang.go`).
2. Open the repo root in VS Code (`File > Open Folder...`).
3. On first open, run **Go: Install/Update Tools** from the command
   palette (`Cmd/Ctrl+Shift+P`) and select all — this installs `gopls`,
   `dlv`, `staticcheck`, etc. used for linting, formatting on save, and
   debugging.

**Recommended workspace settings** (`.vscode/settings.json`):

```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "staticcheck",
  "go.vetOnSave": "package",
  "go.formatTool": "gofmt",
  "editor.formatOnSave": true,
  "go.testFlags": ["-race", "-v"]
}
```

**Running a cluster from VS Code**

Open an integrated terminal per node (`Terminal > New Terminal`, then
split), and start each node as described in
[Starting a local cluster](#starting-a-local-cluster-3-nodes). Running each
node in its own terminal tab makes it easy to watch each node's logs
(peer connections, gossip, block acceptance, reorganisations) side by
side.

**Debugging a single node**

A launch configuration for stepping through one node with breakpoints
(`.vscode/launch.json`):

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch node :8001",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/node",
      "args": ["-addr=:8001", "-peers=:8002,:8003"]
    }
  ]
}
```

Duplicate this block per node (adjusting `-addr` and `-peers`) to debug
several nodes at once from the Run and Debug panel.

**Running tests from VS Code**

Use the **Testing** panel (flask icon) to run/debug individual tests, or
from the integrated terminal:

```bash
go test ./...
go test -race ./...
```

The `go.testFlags` setting above makes the built-in "run test" /
"debug test" CodeLens links above each test function include `-race`
automatically.

## HTTP API

| Method | Path | Description |
|---|---|---|
| GET | `/health` | liveness check |
| GET | `/status` | address, height, head hash, pending count, peers |
| GET | `/height` | current chain height |
| GET | `/chain` | full chain dump (JSON array of blocks) |
| GET | `/peers` | known peer addresses |
| GET | `/balance?address=<pubkey>` | account balance for a given address |
| GET | `/pending` | pending transaction pool size |
| POST | `/transactions` | submit a signed transaction (JSON body) |
| POST | `/blocks` | submit / gossip a mined block (JSON body) |

Example: submit a transaction

```bash
curl -X POST http://localhost:8001/transactions \
  -H "Content-Type: application/json" \
  -d '{"sender":"<pubkey>","receiver":"<pubkey>","amount":10,"publicKey":"...","signature":"..."}'
```

Example: check status

```bash
curl http://localhost:8001/status
```

Both `/transactions` and `/blocks` accept a request from either an
external client or a peer node (identified via the internal
`X-Origin-Peer` header, used to avoid gossiping a message straight back to
where it came from).

## Design decisions

**Concurrency — locks, not channels.** `Node` uses a single
`sync.RWMutex` guarding the chain, pending pool, and peer set. Reads
(`Height`, `HeadHash`, `PeerList`, `Balance`, `ChainSnapshot`) take a read
lock; mutations (`AddBlock`, `AddTransaction`, sync/reorg) take a write
lock. Mining itself runs lock-free: `snapshotForMining` copies what it
needs under a brief read lock, so proof-of-work never blocks gossip or
HTTP handlers while a nonce search is running. This was simpler to reason
about and to verify with `-race` than a channel-based actor model, given
the small, well-defined set of shared fields involved.

**Gossip de-duplication.** Transactions are keyed by their signature,
blocks by their hash, and tracked in `seenTx` / `seenBlocks` maps. A
gossip message that loops back to a node that has already seen it is
dropped silently (the HTTP handler still returns `200 OK`, not an error),
which keeps the forwarding logic simple and prevents infinite gossip
loops around the network.

**Fork resolution.** `chain.ResolveFork` accepts a candidate chain if it
is strictly longer than the current chain, or — at equal length — carries
more cumulative recorded difficulty (`ChainWork`). On acceptance,
`node.SyncFromPeer` diffs the old and new block sets by hash and re-queues
any transaction from a now-orphaned block back into the pending pool
(subject to re-validation against the new ledger state), satisfying FR-6.

**Sync strategy.** A syncing node first asks its peer for its current
height (`GET /height`). If the peer is strictly ahead, the node downloads
and validates the missing blocks one at a time (`GET /blocks/{index}`),
appending each through the same validation path used for gossiped blocks
(`AddBlock` — hash, proof-of-work, Merkle root, previous-hash link, and
ledger replay), matching FR-5's "download the missing blocks, validating
each one." If a fetched block doesn't extend the local tip — meaning the
peer's chain has actually diverged rather than simply run ahead — sync
falls back to fetching and comparing the peer's whole chain via `/chain`,
which is what drives fork resolution under the longest/most-work rule
(FR-6). Reusing `AddBlock` for the common catch-up case means new blocks
are validated with exactly the same rules whether they arrive by gossip
or by sync.

**Signatures and address derivation.** A transaction's sender address is
derived from its signing public key (`SenderAddress()`), not trusted as
free-form input. The ledger additionally binds a sender name to the first
public key it sees signing on that name's behalf, so a second party can't
impersonate an existing sender by signing with a different key pair.

## Known limitations

- `ChainWork` sums per-block `Difficulty` linearly. Real proof-of-work
  difficulty is exponential in the number of leading-zero hex digits
  (each extra digit is roughly 16x harder to satisfy), so this is a
  simplification and not a faithful cumulative-hash-power measure.
- Sync catches up incrementally block-by-block in the common case, but a
  detected fork still falls back to a full `/chain` refetch and compare
  rather than negotiating a common ancestor and downloading only the
  diverging suffix — simpler, at the cost of extra bandwidth on a large,
  deeply forked chain.
- No cross-machine networking — all peers are `localhost:<port>`, as
  specified in scope.
- No hard finality, as with any small proof-of-work chain — see the
  research report for further discussion.

## Tests

```bash
go test ./...
go test -race ./...
```

Test coverage includes signing/verification, gossip de-duplication, chain
sync, fork resolution and reorg, peer health/discovery, persistence
round-trips, and a dedicated concurrency stress test that runs mining,
transaction submission, and reads simultaneously across nodes
(`node/concurrency_test.go`).

## Acceptance criteria

The scenarios below are defined in the assessment spec (Section 9) and map
directly to the functional requirements. The acceptance scenarios are
covered by automated tests where applicable. The multi-node behavior is
also verified through the local cluster demonstration described below.

**A transaction with an invalid signature is rejected (FR-2)**
> Given a node running with a valid chain, when it receives a transaction
> whose signature does not match its public key, then the node rejects
> the transaction and it is not added to the pending pool.

Covered by `node.TestAddTransactionRejectsInvalidSignature`,
`node.TestAddTransactionRejectsUnsignedTransaction`,
`node.TestAddTransactionRejectsWrongPublicKeyForSender`.

**A transaction propagates to a peer (FR-3)**
> Given two connected nodes A and B, when a valid transaction is
> submitted to node A, then the transaction appears in node B's pending
> pool, and it is not forwarded back to node A a second time.

Covered by `node.TestTransactionGossipPropagatesToPeer` and
`node.TestTransactionGossipDeduplicatesAcrossNetwork`.

**A mined block is accepted by a peer (FR-4)**
> Given two connected nodes at the same height, when node A mines and
> broadcasts a new block, then node B validates the block and node B's
> height increases by one.

Covered by `node.TestBlockGossipPropagatesToPeer`,
`node.TestAddBlockAcceptsValidExtendingBlock`,
`node.TestAddBlockRejectsTamperedHash`.

**A new node syncs the chain from a peer (FR-5)**
> Given an existing node with a chain of several blocks, and a new node
> that has only the genesis block, when the new node syncs from the
> existing peer, then the new node's chain matches the existing peer's
> chain.

Covered by `node.TestSyncFromPeerCatchesUpNewNode`,
`node.TestFetchPeerHeightReturnsCurrentHeight`, and
`node.TestFetchPeerBlockReturnsRequestedBlock`.

**The network converges after a fork (FR-6)**
> Given two nodes that each mined a different block at the same height,
> when the two nodes exchange their chains, then both nodes adopt the
> longer valid chain, and any orphaned transactions return to the pending
> pool.

Covered by `node.TestForkConvergenceAndOrphanTransactionRecovery` and, at
the chain-resolution level, `chain.TestResolveForkAcceptsLongerChain`,
`chain.TestResolveForkRejectsShorterChain`,
`chain.TestResolveForkRejectsSameLengthChain`,
`chain.TestResolveForkRejectsInvalidChain`.

**Shared state is free of data races (FR-7)**
> Given a node that is mining while gossiping with peers, when the test
> suite runs with the race detector enabled, then no data race is
> reported.

Covered by `node.TestConcurrentMiningGossipAndReadsAreRaceFree`, run as
part of:

```bash
go test -race ./...
```

## Research report

See [`report.md`](./report.md) for the required experiments (fork
convergence, gossip cost) and the discussion questions (the 51% attack,
finality, and what signatures do and don't prevent).
