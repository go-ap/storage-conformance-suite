package names

import (
	"slices"
	"strings"
	"testing"
)

func TestGetRandom(t *testing.T) {
	for range 100 {
		got := GetRandom()
		pieces := strings.Split(got, "_")
		if len(pieces) != 2 {
			t.Fatalf("invalid name pieces count %d, expected %d", len(pieces), 2)
		}
		leftPiece := pieces[0]
		rightPiece := pieces[1]
		if !slices.Contains(left[:], leftPiece) {
			t.Errorf("GetRandom() left piece was not found in left array = %s", leftPiece)
		}
		if !slices.Contains(right[:], rightPiece) {
			t.Errorf("GetRandom() left piece was not found in left array = %s", leftPiece)
		}
	}
}
