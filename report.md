# Research Report — Toy Blockchain and Ledger Simulator

## 1. Tamper-Evidence

**Setup:** A blockchain was built up over several `add` / `mine` cycles, producing a chain of five blocks (genesis plus four mined blocks). The chain was validated, then the genesis block's stored JSON file (`blockchain.json`) was edited directly — one transaction amount was changed from its original value to a different value — without touching the block's stored `Hash`. The chain was validated again.

**Before tampering:**
========== VALIDATION ==========
Valid: true
Message: Chain is valid

**After tampering (genesis block's transaction amount edited on disk):**
========== VALIDATION ==========
Valid: false
Message: Block 0: hash mismatch (data tampered)

**Why this happens:**

`ValidateChain` first recomputes each block's Merkle root fresh from its current transactions and compares it against the block's stored `MerkleRoot`; it then recomputes the block's hash (which is itself computed over the Merkle root, not the raw transaction list) and compares that against the stored `Hash`. SHA-256 is designed so that changing even one character of input — here, one transaction's `Amount` — produces a completely different, unpredictable output. Because the transaction change alters the Merkle root, and the Merkle root feeds into the block hash, both checks fail together: the stored `MerkleRoot` and `Hash` fields still hold their *original* values, computed before the edit, and neither can be reproduced by recomputing from the tampered data.

This is the very first pair of checks `ValidateChain` performs on every block, including the genesis block, which is why tampering was caught at block 0 specifically — the check does not skip genesis for hash-integrity purposes, only for the previous-hash-link and index checks that don't apply to it.

Critically, editing the transaction amount alone is not enough to produce a *matching* fake hash — an attacker would need to also find a new nonce that makes the recomputed hash satisfy the proof-of-work target again (an expensive search, see Section 2), and then repeat that for every subsequent block, since each block's `PreviousHash` field would also no longer match. A single edited value is caught instantly; producing an edit that passes validation requires redoing the proof-of-work for the tampered block and every block after it.

**A second, independent layer:** with digital signatures now in place, tampering with a transaction is actually caught *earlier* than the block-hash check in some paths. A hand-edited `pending.json` (rather than an already-mined `blockchain.json`) is re-validated at mine time by replaying it against a temporary ledger — and because a transaction's signature is computed over its `Sender`, `Receiver`, and `Amount`, editing any of those fields without re-signing invalidates the signature immediately:
Pending pool contains an invalid transaction, aborting mine: invalid signature for sender Alice

This is a stronger guarantee than hash-integrity alone: even before a tampered transaction is ever mined into a block, it's rejected at the identity layer, because the attacker doesn't hold Alice's private key and can't produce a new valid signature for the edited amount.

---

## 2. Difficulty versus Effort

**Setup:** A single transaction was mined into its own block at difficulties 1 through 5, using the `-difficulty` flag. Nonce (the number of attempts before a valid hash was found) and wall-clock mining time were recorded for each run.

| Difficulty (leading zero hex digits) | Nonce found (attempts) | Mining time |
|---|---|---|
| 1 | 9 | 0 ms (below measurement resolution) |
| 2 | 195 | 0 ms (below measurement resolution) |
| 3 | 9,363 | 10.04 ms |
| 4 | 47,348 | 41.22 ms |
| 5 | 741,027 | 530.45 ms |

**Trend:** The growth is clearly **exponential, not linear**. Going from difficulty 4 to difficulty 5, the nonce count jumped roughly 15.6x (47,348 → 741,027), and mining time jumped roughly 12.9x (41ms → 530ms) — both close to the theoretically expected 16x.

**Why:** A SHA-256 hash, once hex-encoded, is a sequence of characters each drawn from 16 possible values (`0`–`9`, `a`–`f`). Mining is effectively a random search: each attempted nonce produces what behaves like a uniformly random hash. The probability that a given attempt's hash happens to start with exactly `N` zero characters is `1 / 16^N`. So the *expected* number of attempts needed to find a valid nonce is `16^N`:

- Difficulty 1: 16 expected attempts
- Difficulty 2: 256 expected attempts
- Difficulty 3: 4,096 expected attempts
- Difficulty 4: 65,536 expected attempts
- Difficulty 5: 1,048,576 expected attempts

Each additional required leading zero **multiplies** the search space by 16, rather than adding a fixed amount — which is exactly the shape seen in the measured data above (actual nonce counts vary from these theoretical averages because mining is a random process, not a deterministic one — a "lucky" run can find a valid nonce well below the expected count, as happened here at every difficulty level, and an "unlucky" run could take several times longer than expected).

This is the mechanism that makes proof-of-work tunable: a small increase in required difficulty produces a large, controllable increase in the computational cost of mining a block, without changing the algorithm itself — only the target string length. It's also the direct motivation for both **concurrent mining** and **difficulty retargeting** (Sections 3.5 and 3.6): as difficulty grows exponentially expensive, spreading the search across multiple goroutines gives a roughly linear speedup, and automatically adjusting difficulty based on observed block time is what keeps mining from either stalling (too hard) or becoming instant and meaningless (too easy) as conditions change.

---

## 3. Design Write-Up

### 3.1 Hashing scheme

Each block's hash is computed with SHA-256 over a JSON serialization of the following fields, in this fixed order:

1. `Index`
2. `Timestamp`
3. `MerkleRoot`
4. `PreviousHash`
5. `Nonce`
6. `Difficulty`

The `Hash` field itself is deliberately excluded from its own input — a block's hash is a fingerprint *of* the block, so including the fingerprint as an input to itself would be circular and would make the hash trivially unstable. Field order matters because the hash is computed over a serialized byte sequence; the same fields in a different order would produce a different hash for what is conceptually the same block, so the order is fixed and consistent every time a hash is calculated.

Note that the raw `Transactions` slice is *not* one of the hashed fields directly — `MerkleRoot` stands in for it. This is a deliberate design choice explained in 3.2.

`Difficulty` is included deliberately, not just `Nonce` and `MerkleRoot`. This means a block's recorded difficulty cannot be silently altered after the fact — any change to it is caught by the same hash-mismatch check that catches transaction tampering, since it changes the recomputed hash. This also allows different blocks in the same chain to have been legitimately mined at different difficulties (e.g. via the `-difficulty` flag or automatic retargeting), while still being individually verifiable against the difficulty each one actually satisfied.

### 3.2 Merkle root

Rather than hashing the block's transaction list directly as part of the block hash, each block's transactions are first reduced to a single Merkle root: every transaction is hashed individually with SHA-256, then adjacent hashes are concatenated and hashed together in pairs, repeatedly, until one root hash remains (an unpaired final hash at any level is paired with itself, a common convention for handling odd counts). That root is what actually feeds into the block hash.

Two reasons this is worth doing over hashing the raw transaction list directly:

- It decouples "does this list of transactions match what was originally committed" from "how many transactions are there" — the Merkle root has a fixed size regardless of block size, which matters more at scale than in this toy, but is still the conceptually correct structure to reach for.
- It's the standard building block for **Merkle proofs**: proving a single transaction belongs to a block without needing the full transaction list, by supplying only the sibling hashes along the path to the root. This toy computes and verifies the full root rather than individual-transaction proofs (see Known Limitations in the README), but the underlying tree structure is the same one a proof system would be built on top of.

### 3.3 Digital signatures

Every transaction (other than the sender-less genesis coinbase transactions) carries a `PublicKey` and `Signature` field. Signing covers a stable byte encoding of `Sender:Receiver:Amount` — deliberately *not* including `PublicKey` or `Signature` themselves, for the same reason `Hash` excludes itself from a block's hash: signing your own signature would be circular.

Verification happens inside `ledger.ApplyTransaction`, which means it's exercised identically everywhere a transaction is ever applied — adding to the pending pool, mining, and full-chain validation — rather than as a separate check that different code paths might forget to call. Two things are checked together:

1. The signature is cryptographically valid for the given public key and transaction contents (Ed25519 verification).
2. The public key matches the one already registered for this sender *name*, the first time that name was ever seen transacting. This is what actually binds a human-readable name like `"Alice"` to a specific key pair — without it, signatures alone would only prove *some* key pair signed the transaction, not that it was the *same* key pair "Alice" has always used.

### 3.4 Wallet

The wallet is a thin persistence layer mapping names to Ed25519 key pairs (`crypto/ed25519`, standard library). `GetOrCreate("Alice")` either returns Alice's existing key pair or generates and saves a new one — this is the only way key pairs come into existence in the CLI; there's no dedicated `keygen` command, since tying key creation to first use keeps the CLI's existing "sender by name" ergonomics unchanged while making every transaction underneath genuinely signed.

### 3.5 Concurrent mining

Mining is an embarrassingly parallel search: every nonce can be checked independently of every other nonce. `MineBlockConcurrent` exploits this directly — `N` goroutines each search a disjoint arithmetic sequence of nonces (worker `i` tries `i, i+N, i+2N, ...`), each against its own local copy of the block so no shared mutable state is read or written during the hot loop. The only coordination point is a single `atomic.Bool` "found" flag checked at the top of each worker's loop, and a `CompareAndSwap` used exactly once, by whichever worker gets there first, to claim the winning result. This gives close to a linear speedup with core count for the search itself, without needing locks around the hot path.

### 3.6 Difficulty retargeting

`CalculateNextDifficulty` looks back over the last `RetargetInterval` blocks' timestamps and compares actual elapsed time against `RetargetInterval × TargetBlockTimeSeconds`. If blocks came in faster than target, difficulty steps up by `MaxDifficultyStep`; if slower, it steps down — always floored at `MinDifficulty` so it can never trivialize proof-of-work entirely. Retargeting only triggers every `RetargetInterval` blocks rather than every block, which avoids overreacting to the natural variance of a random search (the same variance visible in Section 2's difficulty table). Before enough blocks exist to measure a full interval, `NextDifficultyFor` simply honors whatever difficulty was requested via configuration — this is what lets `-difficulty` meaningfully control the very start of a fresh chain.

### 3.7 Fork resolution

`ResolveFork` implements a longest-valid-chain rule with a work-based tiebreaker, closely mirroring (at a conceptual level) how real proof-of-work chains resolve competing forks. Given a candidate chain, it: (1) rejects an empty or nil candidate outright, (2) fully re-validates the candidate with the same `ValidateChain` used everywhere else, (3) checks the candidate shares the current chain's genesis block hash — rejecting an entirely unrelated chain rather than treating it as a fork of this one, and then (4) compares chain length, falling back to summed per-block difficulty ("work") only if lengths are exactly equal. A losing or invalid candidate leaves the current chain completely untouched; a winning candidate replaces it and is saved to disk by the CLI's `resolve` command.

### 3.8 Validation guarantees

`ValidateChain` walks every block in order and performs, for each:

1. **Merkle root integrity** — recompute the Merkle root from the block's current transactions and compare to the stored `MerkleRoot`. Catches any transaction-level tampering at the earliest possible point.
2. **Hash integrity** — recompute the hash from the block's current fields (including the now-verified Merkle root) and compare to the stored `Hash`. Catches any modification to any field of the block, including nonce or difficulty.
3. **Previous-hash link** (skipped for genesis) — the block's `PreviousHash` must equal the *actual* hash of the prior block. This is what turns a list of independently-hashed blocks into a genuine chain: modifying an old block changes its hash, which then breaks the link recorded in the next block, cascading forward.
4. **Sequential index** (skipped for genesis) — each block's index must be exactly one greater than the previous block's, preventing blocks from being reordered, skipped, or duplicated.
5. **Timestamp ordering** (skipped for genesis) — each block's timestamp must not be earlier than the previous block's, catching obviously invalid or reordered chains.
6. **Minimum difficulty and proof-of-work** — the block's recorded difficulty must be at or above `MinDifficulty`, and the block's hash must actually satisfy that recorded difficulty, so a block cannot claim a difficulty it never genuinely mined at.
7. **Genesis-specific checks** — index must be 0, and previous-hash must equal the fixed all-zero genesis value.
8. **Ledger replay** — every transaction in the block is applied to a running ledger, which itself enforces signature validity, sender identity binding, positive amounts, and sufficient balance.

Validation returns on the *first* block that fails any check, along with a message naming the block index and the specific failure — this makes it possible to know exactly where and how a chain was compromised, rather than just that it was. This same routine is reused in three places: on every load from disk, on every explicit `validate` command, and as the first step of evaluating a fork candidate — so there's exactly one implementation of "is this chain valid" in the whole codebase.

Together, these checks mean that tampering with any block, at any position in the chain, is detectable — and because of the previous-hash chaining, tampering with an *old* block is detectable even without re-validating every subsequent block's proof-of-work from scratch, since the very next block's stored link will already be wrong.

---

## 4. Discussion Questions

### How does the previous-hash link make tampering with an old block impractical in a real chain, even though it is trivial in your local toy?

In this toy, editing an old block is "trivial" only in the sense that nothing stops you from opening `blockchain.json` in a text editor — but the *result* is exactly what would happen in a real chain: the moment a transaction is edited, that block's Merkle root and hash no longer match what's recomputed, and the next block's `PreviousHash` no longer matches the tampered block's new (correct) hash. Making that edit "work" — pass validation — would require re-mining the tampered block (finding a new nonce that satisfies proof-of-work for the new hash) and then re-mining every single block after it, since each one's `PreviousHash` field would also need to be updated and re-mined in turn. On top of that, since transactions are now signed, the attacker would also need the original sender's private key to produce a validly signed replacement transaction in the first place — something editing a JSON file on disk cannot forge.

The difference in a real network is that this re-mining work has to be done *faster than the rest of the network is extending the honest chain*, and the attacker has to control enough hashing power to outpace everyone else combined (the "51% attack" problem). In this toy, there's no competing network — an attacker with a laptop and unlimited time could eventually re-mine everything alone (though not forge signatures, even with unlimited time, given Ed25519's security properties). In a real chain, the same cryptographic mechanisms (hash chaining plus signatures) are what make tampering detectable, but it's the *combination* with a large, competing, honest network that makes it also economically and practically infeasible, not just detectable.

### Proof-of-work is one alternative for deciding who adds the next block. Name at least one alternative and give one advantage and one drawback versus proof-of-work.

**Proof-of-stake (PoS)** is a common alternative: instead of competing to solve a computational puzzle, the right to add the next block is assigned (often semi-randomly) to participants in proportion to how much cryptocurrency they have staked (locked up) in the system.

*Advantage over proof-of-work:* Proof-of-stake requires vastly less energy — there's no large-scale competitive hashing race running continuously, since the "cost" of participating is capital already committed to the system rather than ongoing computation. This was a major motivation for Ethereum's move from proof-of-work to proof-of-stake in 2022.

*Drawback versus proof-of-work:* Proof-of-stake tends to concentrate influence among those who already hold the most stake — the participants with the largest holdings have proportionally the greatest chance of being selected to produce blocks (and earn any associated rewards), which can reinforce existing wealth concentration in the system in a way that's less true of proof-of-work, where influence is tied to hardware and electricity investment rather than pre-existing token holdings.

### List three concrete ways this toy differs from a production blockchain. Pick one and sketch how you would add it.

Having now implemented signatures, Merkle roots, and fork resolution as stretch goals, the most honest remaining gaps are:

1. **No peer-to-peer network or consensus.** This is a single process with one local copy of the chain; fork resolution (Section 3.7) can *evaluate* a competing chain if handed one, but there is no mechanism for multiple independent nodes to discover each other, gossip blocks, or automatically agree on a shared chain state.
2. **No Merkle *proofs* of individual-transaction inclusion.** The Merkle root is computed and verified as a whole (Section 3.2), but there's no API to prove a single transaction belongs to a block using only a handful of sibling hashes, which is the actual efficiency payoff Merkle trees are known for in production systems (e.g. SPV clients).
3. **No mining rewards or transaction fees.** Miners currently have no economic incentive to mine — the only funds in the system are the two fixed genesis coinbase transactions. A production chain needs an incentive structure to make honest mining rational.

**Sketch: adding Merkle proofs.** `block.MerkleRoot` already builds the full tree level-by-level internally but discards everything except the final root. A `MerkleProof(txs []Transaction, index int) []ProofStep` function could instead retain each level and, for a given transaction index, walk back down recording just the sibling hash at each level (and whether it's a left or right sibling, for correct concatenation order). A corresponding `VerifyMerkleProof(txHash string, proof []ProofStep, root string) bool` would recompute the root by hashing the transaction with each sibling in order and check the result matches. This would let a light client hold only block headers (index, timestamp, Merkle root, previous hash, nonce, difficulty, hash — no transaction list) and still verify a specific transaction was included in a specific block, without downloading every transaction in that block.

---

## 5. Honest Notes on Scope

This report and the accompanying implementation cover FR-1 through FR-8 in full, and FR-9 (configurable difficulty, block size, and data file path via command-line flags) as an optional addition. All five stretch goals from Section 13 of the brief are implemented: digital signatures, a Merkle root, concurrent mining, difficulty retargeting, and fork resolution.

Deliberately still out of scope, consistent with Section 4.2 of the assessment brief: networking, peer-to-peer gossip, and distributed consensus between independent nodes; smart contracts or any VM/EVM-style execution; and mining rewards or transaction fees as an economic incentive layer. The goal throughout was a correct, well-tested, and clearly explained system — including the stretch goals — rather than a broader but shallower feature set.