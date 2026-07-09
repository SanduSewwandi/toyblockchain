# Research Report — Toy Blockchain and Ledger Simulator

## 1. Tamper-Evidence

**Setup:** A blockchain was built up over several `add` / `mine` cycles, producing a chain of five blocks (genesis plus four mined blocks). The chain was validated, then the genesis block's stored JSON file (`blockchain.json`) was edited directly — one transaction amount was changed from its original value to a different value — without touching the block's stored `Hash`. The chain was validated again.

**Before tampering:**
```
========== VALIDATION ==========
Valid: true
Message: Chain is valid
```

**After tampering (genesis block's transaction amount edited on disk):**
```
========== VALIDATION ==========
Valid: false
Message: Block 0: hash mismatch (data tampered)
```

**Why this happens:**

`ValidateChain` recomputes each block's hash fresh from its current fields (`Index`, `Timestamp`, `Transactions`, `PreviousHash`, `Nonce`, `Difficulty`) and compares that recomputed value against the block's stored `Hash` field. SHA-256 is designed so that changing even one character of input — here, one transaction's `Amount` — produces a completely different, unpredictable output. The block's `Hash` field still holds the *original* hash, computed before the edit. Once the transaction data changes, recomputing the hash can no longer produce that same original value, so the comparison fails immediately.

This is the very first check `ValidateChain` performs on every block, including the genesis block, which is why tampering was caught at block 0 specifically — the check does not skip genesis for hash-integrity purposes, only for the previous-hash-link and index checks that don't apply to it.

Critically, editing the transaction amount alone is not enough to produce a *matching* fake hash — an attacker would need to also find a new nonce that makes the recomputed hash satisfy the proof-of-work target again (an expensive search, see Section 2), and then repeat that for every subsequent block, since each block's `PreviousHash` field would also no longer match. A single edited value is caught instantly; producing an edit that passes validation requires redoing the proof-of-work for the tampered block and every block after it.

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

This is the mechanism that makes proof-of-work tunable: a small increase in required difficulty produces a large, controllable increase in the computational cost of mining a block, without changing the algorithm itself — only the target string length.

---

## 3. Design Write-Up

### Hashing scheme

Each block's hash is computed with SHA-256 over a JSON serialization of the following fields, in this fixed order:

1. `Index`
2. `Timestamp`
3. `Transactions`
4. `PreviousHash`
5. `Nonce`
6. `Difficulty`

The `Hash` field itself is deliberately excluded from its own input — a block's hash is a fingerprint *of* the block, so including the fingerprint as an input to itself would be circular and would make the hash trivially unstable. Field order matters because the hash is computed over a serialized byte sequence; the same fields in a different order would produce a different hash for what is conceptually the same block, so the order is fixed and consistent every time a hash is calculated.

`Difficulty` is included deliberately, not just `Nonce` and `Transactions`. This means a block's recorded difficulty cannot be silently altered after the fact — any change to it is caught by the same hash-mismatch check that catches transaction tampering, since it changes the recomputed hash. This also allows different blocks in the same chain to have been legitimately mined at different difficulties (e.g. via the `-difficulty` flag), while still being individually verifiable against the difficulty each one actually satisfied.

### Validation guarantees

`ValidateChain` walks every block in order and performs, for each:

1. **Hash integrity** — recompute the hash from the block's current fields and compare to the stored `Hash`. Catches any modification to any field of the block, including transactions, nonce, or difficulty.
2. **Previous-hash link** (skipped for genesis) — the block's `PreviousHash` must equal the *actual* hash of the prior block. This is what turns a list of independently-hashed blocks into a genuine chain: modifying an old block changes its hash, which then breaks the link recorded in the next block, cascading forward.
3. **Sequential index** (skipped for genesis) — each block's index must be exactly one greater than the previous block's, preventing blocks from being reordered, skipped, or duplicated.
4. **Timestamp ordering** (skipped for genesis) — each block's timestamp must not be earlier than the previous block's, catching obviously invalid or reordered chains.
5. **Proof-of-work** — the block's hash must actually satisfy its own recorded difficulty target, so a block cannot claim a difficulty it never genuinely mined at.
6. **Genesis-specific checks** — index must be 0, and previous-hash must equal the fixed all-zero genesis value.

Validation returns on the *first* block that fails any check, along with a message naming the block index and the specific failure — this makes it possible to know exactly where and how a chain was compromised, rather than just that it was.

Together, these checks mean that tampering with any block, at any position in the chain, is detectable — and because of the previous-hash chaining, tampering with an *old* block is detectable even without re-validating every subsequent block's proof-of-work from scratch, since the very next block's stored link will already be wrong.

---

## 4. Discussion Questions

### How does the previous-hash link make tampering with an old block impractical in a real chain, even though it is trivial in your local toy?

In this toy, editing an old block is "trivial" only in the sense that nothing stops you from opening `blockchain.json` in a text editor — but the *result* is exactly what would happen in a real chain: the moment a transaction is edited, that block's hash no longer matches what's recomputed, and the next block's `PreviousHash` no longer matches the tampered block's new (correct) hash. Making that edit "work" — pass validation — would require re-mining the tampered block (finding a new nonce that satisfies proof-of-work for the new hash) and then re-mining every single block after it, since each one's `PreviousHash` field would also need to be updated and re-mined in turn.

The difference in a real network is that this re-mining work has to be done *faster than the rest of the network is extending the honest chain*, and the attacker has to control enough hashing power to outpace everyone else combined (the "51% attack" problem). In this toy, there's no competing network — an attacker with a laptop and unlimited time could eventually re-mine everything alone. In a real chain, the same cryptographic mechanism (hash chaining) is what makes tampering detectable, but it's the *combination* with a large, competing, honest network that makes it also economically and practically infeasible, not just detectable.

### Proof-of-work is one alternative for deciding who adds the next block. Name at least one alternative and give one advantage and one drawback versus proof-of-work.

**Proof-of-stake (PoS)** is a common alternative: instead of competing to solve a computational puzzle, the right to add the next block is assigned (often semi-randomly) to participants in proportion to how much cryptocurrency they have staked (locked up) in the system.

*Advantage over proof-of-work:* Proof-of-stake requires vastly less energy — there's no large-scale competitive hashing race running continuously, since the "cost" of participating is capital already committed to the system rather than ongoing computation. This was a major motivation for Ethereum's move from proof-of-work to proof-of-stake in 2022.

*Drawback versus proof-of-work:* Proof-of-stake tends to concentrate influence among those who already hold the most stake — the participants with the largest holdings have proportionally the greatest chance of being selected to produce blocks (and earn any associated rewards), which can reinforce existing wealth concentration in the system in a way that's less true of proof-of-work, where influence is tied to hardware and electricity investment rather than pre-existing token holdings.

### List three concrete ways this toy differs from a production blockchain. Pick one and sketch how you would add it.

1. **No peer-to-peer network or consensus.** This is a single process with one local copy of the chain; there is no mechanism for multiple independent nodes to agree on a shared chain state, and no way to resolve disagreements between them.
2. **No transaction signatures.** Transactions here carry a plain sender name with no cryptographic proof that the named sender actually authorized them — anyone could construct a transaction claiming to be from "Alice" with no way to verify that.
3. **No Merkle tree.** Each block's hash currently depends on the *entire* transaction list being hashed directly; there is no compact way to prove a single transaction is included in a block without transmitting and hashing the entire block's transaction list.

**Sketch: adding transaction signatures.** Each account would be represented by a public/private key pair instead of a plain name string. A `Transaction` would gain a `Signature` field, produced by signing a hash of the transaction's `Sender`, `Receiver`, and `Amount` with the sender's private key (Go's standard library `crypto/ecdsa` or `crypto/ed25519` would cover this without external dependencies). `Ledger.ApplyTransaction` would be extended to first verify the signature against the sender's known public key before applying any balance change, rejecting the transaction if verification fails — the same rejection pattern already used for non-positive amounts and insufficient balance. The `Sender` field would effectively become a public key (or an address derived from one) rather than an arbitrary string, which would also incidentally close a gap in the current implementation: right now, nothing stops one user from spending from another user's named account.

---

## 5. Honest Notes on Scope

This report and the accompanying implementation cover FR-1 through FR-8 in full, and FR-9 (configurable difficulty, block size, and data file path via command-line flags) as an optional addition. Networking, consensus, digital signatures, Merkle trees, smart contracts, and fork resolution were all deliberately left out of scope, consistent with Section 4.2 of the assessment brief — the goal here was a correct, well-tested, and clearly explained core rather than a broader but shallower feature set.
