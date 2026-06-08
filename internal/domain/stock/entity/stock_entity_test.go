package entity

import "testing"

func TestSignedQuantity(t *testing.T) {
	cases := []struct {
		name         string
		movementType string
		qty          float64
		want         float64
	}{
		{"in is positive", MovementTypeIn, 10, 10},
		{"transfer in is positive", MovementTypeTransferIn, 5, 5},
		{"legacy ENTRADA is positive", "ENTRADA", 7, 7},
		{"out is negative", MovementTypeOut, 10, -10},
		{"transfer out is negative", MovementTypeTransferOut, 4, -4},
		{"legacy SAIDA is negative", "SAIDA", 3, -3},
		{"adjustment keeps sign", MovementTypeAdjustment, -2, -2},
		{"reservation does not affect on-hand", "RESERVATION", 9, 0},
		{"unknown type does not affect on-hand", "FOO", 9, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SignedQuantity(tc.movementType, tc.qty); got != tc.want {
				t.Fatalf("SignedQuantity(%q, %v) = %v, want %v", tc.movementType, tc.qty, got, tc.want)
			}
		})
	}
}
