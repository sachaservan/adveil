package token

import (
	"crypto/elliptic"
	"crypto/rand"
	_ "crypto/sha256"
	"testing"

	"github.com/sachaservan/adveil/crypto"
)

func TestBlindingP256(t *testing.T) {
	curve := elliptic.P256()
	_, x, y, _ := elliptic.GenerateKey(curve, rand.Reader)
	X := &crypto.Point{Curve: curve, X: x, Y: y}
	P, r := Blind(X)
	Xprime := Unblind(P, r)
	if X.X.Cmp(Xprime.X) != 0 || X.Y.Cmp(Xprime.Y) != 0 {
		t.Fatal("unblinding failed to produce the same point")
	}
}

func TestFullProtocol(t *testing.T) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	// Client: generate and store (token, bF, bP)
	token, bP, bF, err := pk.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	// 2b. Sign the blind point
	sigBlind := sk.Sign(bP)

	// Client: unblind signature
	sig := Unblind(sigBlind, bF)

	signed := &SignedToken{token, sig}

	// Server: redeem unblinded token and signature
	valid := sk.Redeem(signed)
	if !valid {
		t.Fatal("failed redemption")
	}
}

func TestFullProtocolWithProof(t *testing.T) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	// Client: generate and store (token, bF, bP)
	token, bP, bF, err := pk.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	// 2b. Sign the blind point
	sigBlind := sk.Sign(bP)

	// Client: unblind signature
	sig := Unblind(sigBlind, bF)

	signed := &SignedToken{token, sig}

	// Server: redeem unblinded token and signature
	valid, proof := sk.RedeemAndProve(pk, signed)
	if !valid {
		t.Fatal("failed redemption")
	}

	h2cObject, _ := crypto.GetDefaultCurveHash()
	if !proof.Verify(h2cObject) {
		t.Fatal("failed proof verification")
	}
}

func BenchmarkBlinding(b *testing.B) {
	_, X, err := crypto.NewRandomPoint()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		Blind(X)
	}
}

func BenchmarkUnblinding(b *testing.B) {
	_, X, err := crypto.NewRandomPoint()
	if err != nil {
		panic(err)
	}

	P, r := Blind(X)
	if P == nil || r == nil {
		b.Fatalf("nil ret values")
	}

	for i := 0; i < b.N; i++ {
		Unblind(P, r)
	}
}
