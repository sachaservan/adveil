package elgamal

import (
	"crypto/elliptic"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	expected := pk.RandomPoint()
	actual := sk.Decrypt(pk.Encrypt(expected))

	if expected.X.Cmp(actual.X) != 0 || expected.Y.Cmp(actual.Y) != 0 {
		t.Fatal("encryption decryption error")
	}
}
