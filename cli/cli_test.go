package cli

import "testing"

func TestIsValidSender(t *testing.T) {

	cases := map[string]bool{
		"Alice":  true,
		"Bob":    true,
		"":       false,
		"SYSTEM": false,
		"system": false,
		"System": false,
	}

	for sender, want := range cases {

		if got := isValidSender(sender); got != want {

			t.Errorf(
				"isValidSender(%q) = %v, want %v",
				sender,
				got,
				want,
			)
		}
	}
}