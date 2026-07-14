package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"toyblockchain/ledger"
)

func MerkleRoot(txs []ledger.Transaction) string {

	if len(txs) == 0 {

		empty := sha256.Sum256([]byte{})
		return hex.EncodeToString(empty[:])
	}

	var level []string

	for _, tx := range txs {

		data, err := json.Marshal(tx)
		if err != nil {
			panic(err)
		}

		h := sha256.Sum256(data)
		level = append(level, hex.EncodeToString(h[:]))
	}

	for len(level) > 1 {

		var next []string

		for i := 0; i < len(level); i += 2 {

			left := level[i]
			right := left

			if i+1 < len(level) {
				right = level[i+1]
			}

			h := sha256.Sum256([]byte(left + right))
			next = append(next, hex.EncodeToString(h[:]))
		}

		level = next
	}

	return level[0]
}
