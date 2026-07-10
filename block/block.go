package block

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"toyblockchain/ledger"
)

type Block struct {
	Index        int
	Timestamp    int64
	Transactions []ledger.Transaction
	PreviousHash string
	Nonce        int
	Hash         string
	Difficulty   int
}

func (b *Block) CalculateHash() string {

	data := struct {
		Index        int
		Timestamp    int64
		Transactions []ledger.Transaction
		PreviousHash string
		Nonce        int
		Difficulty   int
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PreviousHash: b.PreviousHash,
		Nonce:        b.Nonce,
		Difficulty:   b.Difficulty,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(jsonBytes)

	return hex.EncodeToString(hash[:])
}

func NewBlockWithTimestamp(index int, txs []ledger.Transaction, previousHash string, difficulty int, timestamp int64) Block {

	newBlock := Block{
		Index:        index,
		Timestamp:    timestamp,
		Transactions: txs,
		PreviousHash: previousHash,
		Nonce:        0,
		Difficulty:   difficulty,
	}

	newBlock.Hash = newBlock.CalculateHash()

	return newBlock
}

func NewBlock(index int, txs []ledger.Transaction, previousHash string, difficulty int) Block {

	return NewBlockWithTimestamp(index, txs, previousHash, difficulty, time.Now().Unix())
}
