package token

import (
	"crypto/elliptic"
	_ "crypto/sha256"
	"testing"
)

func TestVanillaTokenProtocol(t *testing.T) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	// Client: generate and store (token, bF, bP)
	token, uT, u, v, err := pk.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	uW := sk.Sign(uT, true)

	// Client: unblind signature
	W := pk.Unblind(uW, token, u, v)

	// Server: redeem unblinded token and signature
	valid := sk.Redeem(W)
	if !valid {
		t.Fatal("failed redemption")
	}
}

func TestPublicMDTokenProtocol(t *testing.T) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)

	// Client: generate and store (token, bF, bP)
	token, uT, u, err := pk.NewPublicMDToken()
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	uW := sk.PublicMDSign(uT, metadata)

	// Client: unblind signature
	W, err := pk.PublicMDUnblind(uW, token, u)

	if err != nil {
		t.Fatal(err)
	}

	// Server: redeem unblinded token and signature
	valid := sk.PublicMDRedeem(W, metadata)
	if !valid {
		t.Fatal("failed redemption")
	}
}

func BenchmarkGenToken(b *testing.B) {

	curve := elliptic.P256()
	pk, _ := KeyGen(curve)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.NewToken()
	}

}

func BenchmarkGenPublicMDToken(b *testing.B) {

	curve := elliptic.P256()
	pk, _ := KeyGen(curve)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.NewPublicMDToken()
	}
}

func BenchmarkPublicMDTokenUnblind(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)
	t, uT, u, _ := pk.NewPublicMDToken()

	// Server: sign blinded token
	uW := sk.PublicMDSign(uT, metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.PublicMDUnblind(uW, t, u)
	}
}

func BenchmarkPublicMDTokenSign(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)

	_, uT, _, err := pk.NewPublicMDToken()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.PublicMDSign(uT, metadata)
	}

}

func BenchmarkPublicMDTokenRedeem(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)

	// Client: generate and store (token, bF, bP)
	t, uT, u, err := pk.NewPublicMDToken()
	if err != nil {
		b.Fatal(err)
	}

	uW := sk.PublicMDSign(uT, metadata)

	W, err := pk.PublicMDUnblind(uW, t, u)

	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.PublicMDRedeem(W, metadata)
	}
}

func BenchmarkTokenSign(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	_, uT, _, _, err := pk.NewToken()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Sign(uT, true)
	}

}

func BenchmarkTokenRedeem(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	t, uT, u, v, err := pk.NewToken()
	if err != nil {
		b.Fatal(err)
	}

	uW := sk.Sign(uT, true)

	W := pk.Unblind(uW, t, u, v)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Redeem(W)
	}
}
