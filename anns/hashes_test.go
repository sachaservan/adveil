package anns

import (
	"testing"

	"github.com/ncw/gmp"
)

func TestUniversalHashBuild(t *testing.T) {
	for i := 2; i < 100; i++ {
		NewUniversalHash(i)
	}
}

func TestUniversalHashDigestSize(t *testing.T) {
	for i := 2; i < 100; i++ {
		h := NewUniversalHash(i)
		res := h.Digest(gmp.NewInt(int64(i)))

		if len(res.Bytes()) > i {
			t.Fatalf("Wrong digest size. Expected %v, got: %v", i, len(res.Bytes()))
		}
	}
}
