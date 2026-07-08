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
}


func (b *Block) CalculateHash() string {

	data := struct {
		Index        int
		Timestamp    int64
		Transactions []ledger.Transaction
		PreviousHash string
		Nonce        int
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		Transactions: b.Transactions,
		PreviousHash: b.PreviousHash,
		Nonce:        b.Nonce,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(jsonBytes)

	return hex.EncodeToString(hash[:])
}


func NewBlock(index int, txs []ledger.Transaction, previousHash string) Block {

	newBlock := Block{
		Index:        index,
		Timestamp:    time.Now().Unix(),
		Transactions: txs,
		PreviousHash: previousHash,
		Nonce:        0,
	}

	newBlock.Hash = newBlock.CalculateHash()

	return newBlock
}
