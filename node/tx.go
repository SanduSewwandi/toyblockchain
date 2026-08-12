package node

import (
	"fmt"

	"toyblockchain/ledger"
)

func (n *Node) AddTransaction(tx ledger.Transaction) (accepted bool, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Ignore duplicate transactions.
	// The transaction signature is used as the transaction identifier.
	if n.seenTx[tx.Signature] {
		return false, nil
	}

	ld := n.pendingLedgerLocked()

	// Validate the new transaction against the resulting state.
	if err := ld.ApplyTransaction(tx); err != nil {
		return false, fmt.Errorf("transaction rejected: %w", err)
	}

	// The transaction is valid, so add it to the pending pool.
	n.Pending = append(n.Pending, tx)

	// Mark the transaction as seen so repeated gossip messages are ignored.
	n.seenTx[tx.Signature] = true

	return true, nil
}

func (n *Node) removeMinedTransactionsLocked(
	mined []ledger.Transaction,
) {
	minedSigs := make(map[string]bool, len(mined))

	for _, tx := range mined {
		minedSigs[tx.Signature] = true
	}

	remaining := n.Pending[:0]

	for _, tx := range n.Pending {
		if !minedSigs[tx.Signature] {
			remaining = append(remaining, tx)
		}
	}

	n.Pending = remaining
}

func (n *Node) pendingLedgerLocked() *ledger.Ledger {
	ld := n.Blockchain.BuildLedger()

	for _, tx := range n.Pending {
		// Pending transactions were validated when they were added.
		// Therefore an error here indicates an unexpected internal state.
		_ = ld.ApplyTransaction(tx)
	}

	return ld
}
