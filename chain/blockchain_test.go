package chain

import (
	"testing"

)


// Existing tests ...


func TestGenesisCreatesInitialBalances(t *testing.T) {

	bc := NewBlockchain()

	ld := bc.BuildLedger()


	if ld.GetBalance("Alice") != 100 {

		t.Errorf(
			"expected Alice balance 100, got %.2f",
			ld.GetBalance("Alice"),
		)
	}


	if ld.GetBalance("Bob") != 50 {

		t.Errorf(
			"expected Bob balance 50, got %.2f",
			ld.GetBalance("Bob"),
		)
	}
}