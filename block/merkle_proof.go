package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"toyblockchain/ledger"
)

// MerkleProofNode is one step in an inclusion proof: the sibling hash
// needed at that level, and which side it sits on relative to the hash
// being carried up from the leaf.
type MerkleProofNode struct {
	Hash     string
	Position string // "left" or "right"
}

// MerkleProof is everything needed to prove a single transaction was
// included in a block's Merkle tree, without needing the block's full
// transaction list.
type MerkleProof struct {
	TxIndex int
	TxHash  string
	Root    string
	Path    []MerkleProofNode
}

// leafHashes reproduces MerkleRoot's leaf-hashing step exactly, so a
// generated proof always matches the root actually stored on the block.
func leafHashes(txs []ledger.Transaction) []string {

	hashes := make([]string, 0, len(txs))

	for _, tx := range txs {

		data, err := json.Marshal(tx)
		if err != nil {
			panic(err)
		}

		h := sha256.Sum256(data)
		hashes = append(hashes, hex.EncodeToString(h[:]))
	}

	return hashes
}

// GenerateMerkleProof builds an inclusion proof for the transaction at
// index within txs, walking the same tree MerkleRoot builds and
// recording the sibling hash needed at each level.
func GenerateMerkleProof(txs []ledger.Transaction, index int) (MerkleProof, error) {

	if index < 0 || index >= len(txs) {
		return MerkleProof{}, fmt.Errorf(
			"transaction index %d out of range (block has %d transactions)",
			index,
			len(txs),
		)
	}

	level := leafHashes(txs)
	txHash := level[index]

	var path []MerkleProofNode
	idx := index

	for len(level) > 1 {

		var next []string

		for i := 0; i < len(level); i += 2 {

			left := level[i]
			right := left // odd node duplicates itself, matching MerkleRoot

			if i+1 < len(level) {
				right = level[i+1]
			}

			h := sha256.Sum256([]byte(left + right))
			next = append(next, hex.EncodeToString(h[:]))

			switch idx {
			case i:
				path = append(path, MerkleProofNode{Hash: right, Position: "right"})
			case i + 1:
				path = append(path, MerkleProofNode{Hash: left, Position: "left"})
			}
		}

		idx = idx / 2
		level = next
	}

	return MerkleProof{
		TxIndex: index,
		TxHash:  txHash,
		Root:    level[0],
		Path:    path,
	}, nil
}

// VerifyMerkleProof recomputes the root from the leaf hash and sibling
// path, and checks it matches the claimed root — the actual proof
// check a verifier (a light client, or a test) would run.
func VerifyMerkleProof(proof MerkleProof) bool {

	current := proof.TxHash

	for _, node := range proof.Path {

		var combined string

		if node.Position == "right" {
			combined = current + node.Hash
		} else {
			combined = node.Hash + current
		}

		h := sha256.Sum256([]byte(combined))
		current = hex.EncodeToString(h[:])
	}

	return current == proof.Root
}
