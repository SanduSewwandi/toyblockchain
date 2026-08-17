# Research Report — Round 2: Networked Multi-Node Blockchain

## 1. Convergence after a fork

**Setup.** Two nodes, A (`:8001`) and B (`:8002`), start from the same
genesis block and are *not* pointed at each other's peer list, so they
cannot gossip.

```bash
go run ./cmd/node -addr=:8001 -peers="" -no-mine
go run ./cmd/node -addr=:8002 -peers="" -no-mine
```

A transaction is submitted to each node separately so each has something
to mine:

```bash
curl -X POST http://localhost:8001/transactions -H "Content-Type: application/json" -d '{...}'
curl -X POST http://localhost:8002/transactions -H "Content-Type: application/json" -d '{...}'
```

Each node then mines one block from its own pending pool (via its mining
loop, or manually triggered), producing two different blocks at height 1
that both extend the same genesis block — a fork.

**Before reconnecting**, chain state on each node:

```
node A: height=1 head=<hash_A_1>
node B: height=1 head=<hash_B_1>
```

_TODO: paste actual `/status` output from both nodes here._

**Reconnecting.** B is added to A's peer list (or vice versa) and a sync
is triggered:

```bash
curl http://localhost:8001/peers
# ... trigger sync, e.g. restart with -peers=:8002 or call SyncFromPeer directly
```

**After reconnecting**, both nodes converge:

```
node A: height=<N> head=<hash>
node B: height=<N> head=<hash>
```

_TODO: paste actual post-sync `/status` output from both nodes, showing
matching height and head hash._

**Which block was orphaned, and what happened to its transactions.**

The losing fork's block (whichever of the two didn't win — record which
here, e.g. "node B's block at height 1 was orphaned because node A's
chain was adopted") is discarded from the active chain. Per
`node.SyncFromPeer`, any block present in the old chain but absent from
the new chain has its transactions re-queued into the pending pool
(subject to re-validation against the new chain's ledger state via
`reconcilePendingLocked`). In this run, the orphaned block's transaction
was: _TODO — state the transaction and confirm it reappeared in the
losing node's `/pending` count._

This matches `node.TestForkConvergenceAndOrphanTransactionRecovery`,
which automates exactly this scenario and asserts the orphaned
transaction returns to the pending pool.

**Why this rule was chosen.** `chain.ResolveFork` accepts a candidate
chain if it is strictly longer, or — at equal length — has more
cumulative `ChainWork` (see Design Write-up below). This is the standard
longest-valid-chain heuristic: it gives independent nodes a simple,
deterministic rule to agree on without a coordinator, at the cost of only
probabilistic (not immediate) agreement — discussed further in Section 3
below.

---

## 2. Gossip cost

**Setup.** Three nodes, A, B, C, fully connected to each other (a
triangle: each peer list contains the other two).

```bash
go run ./cmd/node -addr=:8001 -peers=:8002,:8003
go run ./cmd/node -addr=:8002 -peers=:8001,:8003
go run ./cmd/node -addr=:8003 -peers=:8001,:8002
```

**Measurement.** One transaction is submitted to node A. Each node logs
every inbound `POST /transactions` it receives (see `loggingMiddleware`
in `node/server.go`), so the number of times the endpoint is hit across
all three nodes for this single transaction can be counted directly from
the logs.

```bash
curl -X POST http://localhost:8001/transactions -H "Content-Type: application/json" -d '{...}'
```

_TODO: paste the relevant log lines from all three nodes' terminals,
showing each `POST /transactions` hit, and count them._

**Expected shape of the result.** With de-duplication (`seenTx`, keyed by
transaction signature):

- A receives the transaction once (from the client) → forwards to B and C
  → 2 outbound gossip calls.
- B and C each receive it once, mark it seen, and forward to their peers
  (which includes each other and A) → but since A, and each other, will
  already have marked the signature as seen by the time it loops back,
  those re-deliveries are accepted as no-ops (`"transaction already seen"`,
  HTTP 200) rather than being re-gossiped further.

So for a fully connected 3-node network the total message count is
bounded (roughly `O(N²)` for `N` nodes in the worst case of a complete
graph — each node forwards once to each of its `N-1` peers, and the
receiving nodes' de-duplication stops any further re-broadcast), not
unbounded. _TODO: replace with your actual observed count once you've run
this and counted log lines — the real number is what belongs in a report
like this, not just the theoretical bound._

**How de-duplication stops this from exploding.** Every node tracks
transaction signatures it has already accepted in `seenTx`. `AddTransaction`
returns `accepted=false, err=nil` for anything already in that map, and
`handleTransactions` responds `200 OK` with `"reason": "transaction already
seen"` instead of re-gossiping. This means a message can travel around a
cycle in the peer graph at most once per edge before every node has seen
it and stops forwarding — without this check, a fully connected mesh would
gossip the same transaction indefinitely.

---

## 3. Design write-up

### Wire format and endpoints

All node-to-node and client-to-node communication is plain HTTP with JSON
bodies — no custom binary protocol or serialization. See the README's
[HTTP API](./README.md#http-api) table for the full endpoint list. The two
gossip-carrying endpoints are `POST /transactions` and `POST /blocks`,
both of which:

- decode the JSON body into a `ledger.Transaction` or `block.Block`,
- validate it fully before accepting,
- forward it to the node's other peers via the same JSON POST format,
  tagging the request with an `X-Origin-Peer` header so the receiving
  peer can skip gossiping it straight back to where it came from.

Choosing plain HTTP/JSON over a custom binary protocol trades some
bandwidth efficiency for being trivially debuggable with `curl`, readable
in logs, and requiring no protocol-specific tooling — reasonable given
the scale (a handful of local nodes) and the assessment's "simple HTTP
endpoints and logs are enough" scope note.

Chain synchronisation (FR-5) uses two further read endpoints:
`GET /height`, so a syncing node can see how far behind it is, and
`GET /blocks/{index}`, so it can fetch and validate the missing blocks
one at a time rather than requesting the whole chain. `GET /chain` (a
full chain dump) is retained separately and used only as the
fork-resolution fallback — see below.

### Race-free shared state: locks vs channels

`Node` holds all shared, mutable state (`Blockchain`, `Pending`, `Peers`,
`seenTx`, `seenBlocks`) behind a single `sync.RWMutex`. Read-only
operations (`Height`, `HeadHash`, `PeerList`, `Balance`, `PendingCount`,
`ChainSnapshot`) take a read lock; anything that mutates state
(`AddBlock`, `AddTransaction`, `SyncFromPeer`) takes a write lock.

This was chosen over a channel-based / actor-style design (a single
goroutine owning state, with all access mediated through channels)
because:

- the set of shared fields is small and well-defined, so a coarse mutex
  is easy to reason about correctness-wise;
- the actual proof-of-work loop — the one genuinely expensive operation —
  is explicitly kept *outside* the lock (`snapshotForMining` copies just
  what mining needs under a brief read lock, then mines lock-free), so
  the mutex is never held for the duration of an expensive computation;
- it maps directly onto Go's `net/http` handler model, where each request
  is already its own goroutine — a mutex fits that concurrency shape more
  naturally than introducing a separate coordinating goroutine and
  request/response channels for every operation.

Correctness of this approach is verified with `go test -race ./...`
(clean pass across all packages), plus a dedicated stress test,
`TestConcurrentMiningGossipAndReadsAreRaceFree`, which runs mining,
transaction submission (both valid and deliberately invalid/rejected),
and reads against every introspection endpoint simultaneously across two
nodes.

### How a reorganisation rebuilds the ledger

The ledger is never stored independently — it's always derived fresh from
the chain via `Blockchain.BuildLedger()`, which replays every
transaction in every block from genesis forward. This means a
reorganisation doesn't need to "undo" balance changes block by block;
switching `n.Blockchain.Blocks` to the new chain and rebuilding the
ledger from that new block list is sufficient and correct by
construction — there's no separate mutable balance state to reconcile.

The remaining work in `SyncFromPeer` is recovering *pending* transactions
that were confirmed in a now-orphaned block:

1. Snapshot the current (about-to-be-replaced) block list.
2. Call `ResolveFork`, which — if the candidate wins — replaces
   `n.Blockchain.Blocks` in place.
3. Diff the old block list against the new one by hash; any block in the
   old list that isn't in the new list is orphaned.
4. Re-queue every transaction from orphaned blocks back into `n.Pending`.
5. Call `reconcilePendingLocked`, which rebuilds a ledger from the *new*
   chain, replays the pending pool against it, and drops anything that's
   already confirmed in the new chain or that no longer applies cleanly
   (e.g. now double-spent) — so only transactions that are both new and
   still valid survive back into the pool.

---

## 4. Discussion questions

### Why does the longest-valid-chain rule only work probabilistically? What is a 51% attack?

The longest-chain rule doesn't guarantee any single block is permanent —
it only says that, among the chains a node currently knows about, the
longest one wins. A competing chain that's currently shorter can still
overtake the adopted chain later if it grows faster. Agreement is
therefore never certain, only increasingly *likely* the further back in
history a block sits, because overtaking an old block means out-mining
the entire honest network from that point forward — which becomes
exponentially less probable the more blocks have been added on top of it.

A 51% attack is what happens when a single party controls more than half
of the network's total mining power. With a majority of hashing power,
that party can consistently out-mine everyone else, allowing them to
build a longer private chain than the honest chain and later "reveal" it
to force a reorganisation — enabling double-spends (reversing transactions
they already got real-world value for) or refusing to include certain
transactions. It doesn't let them forge signatures or steal others' funds
outright, but it does let them rewrite recent history on the chain they
control the majority of. This toy network is trivially "51%-attackable"
since it's only a handful of nodes with no economic cost to mining, but
the same mechanism underlies real security discussions around small
proof-of-work networks.

### What does finality mean, and why does a small proof-of-work chain never offer hard finality? How do real networks reduce the risk?

Finality means a transaction, once included in the chain, is guaranteed
never to be reversed. Proof-of-work chains never offer *hard* finality —
only probabilistic finality — because any block can, in principle, always
be displaced by a longer competing chain arriving later; `ResolveFork` in
this project will happily replace the current chain the moment a longer
valid one shows up, no matter how "settled" the current chain looked a
moment before.

Real networks reduce this risk mainly by waiting for confirmations: rather
than treating a transaction as final the instant its block is mined,
recipients wait for several additional blocks to be built on top of it.
Since overtaking a chain requires out-mining the honest network from that
point forward, the probability of a successful reorg shrinks rapidly with
each additional confirming block. This project's node doesn't implement a
confirmation-count concept — any accepted block is immediately reflected
in `/balance` and `/status` — which is a reasonable simplification for a
local toy network but a real limitation relative to production chains.

### What do signatures prevent? What could a malicious peer still try, and how does this network defend against it?

Signatures prevent forgery and impersonation at the point a transaction
enters the network: because `ledger.ApplyTransaction` requires a valid
ed25519 signature over the transaction's contents, and because the
sender's address is derived from — and the ledger binds a sender name to
— the actual signing public key (`SenderAddress()`,
`registeredKeys` in `ledger.Ledger`), nobody can submit a transaction
spending funds from an account they don't hold the private key for, and
nobody can impersonate an already-active sender name with a different
key pair.

What signatures don't prevent is a malicious peer withholding or
selectively delaying gossip, or attempting to feed a node a competing
chain built entirely at minimum difficulty in the hope it gets accepted.
The second is a real, if now narrower, gap in this implementation: a
single block arriving via gossip or via incremental sync (`AddBlock`)
*is* checked against the expected retargeted difficulty
(`NextDifficultyFor`), not just its own claimed value — so a low-effort
block can't sneak in through the normal catch-up path. However, the
fork-resolution fallback (`syncFullChainFromPeer` → `ValidateChain`) only
checks that each block's hash satisfies *its own claimed* difficulty, not
that the claimed difficulty matches what retargeting would have required
at that point in history. So a malicious peer could still, in principle,
present a full alternate chain mined entirely at `MinDifficulty` and have
it evaluated purely on length/work during fork resolution. Hardening
`chain.ValidateChain` to recompute and check expected difficulty per
block (not just each block's self-consistency) would close this
remaining gap, and is a natural next step beyond the current
implementation.

---

*Sources: general blockchain/proof-of-work concepts referenced above
(51% attacks, probabilistic finality, confirmation counts) reflect
standard, widely available descriptions of how proof-of-work consensus
works (e.g. as popularized by Bitcoin's design) rather than any single
cited source; no material was reproduced from any specific external
text.*
