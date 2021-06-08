package token

import (
	"crypto/elliptic"
	_ "crypto/sha256"
	"testing"
)

func TestPrivateMetadataBitProtocol(t *testing.T) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	// Client: generate and store (token, bF, bP)
	token, uT, u, v, err := pk.NewPrivateMDToken()
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	uW := sk.PrivateMDSign(uT, true)

	// Client: unblind signature
	W := pk.PrivateMDUnblind(uW, u, v)

	signed := &SignedToken{token, W, nil, nil}

	// Server: redeem unblinded token and signature
	valid := sk.PrivateMDRedeem(signed)
	if !valid {
		t.Fatal("failed redemption")
	}
}

func TestPublicMetadataProtocol(t *testing.T) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)

	// Client: generate and store (token, bF, bP)
	token, uT, u, err := pk.NewPublicMDToken(metadata)
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	uW, proof := sk.PublicMDSign(uT, metadata)

	// Client: unblind signature
	W, err := pk.PublicMDUnblind(uW, u, metadata, proof)

	if err != nil {
		t.Fatal(err)
	}

	signed := &SignedToken{token, W, nil, nil}

	// Server: redeem unblinded token and signature
	valid := sk.PublicMDRedeem(signed, metadata)
	if !valid {
		t.Fatal("failed redemption")
	}
}

func BenchmarkGenTokenPrivate(b *testing.B) {

	curve := elliptic.P256()
	pk, _ := KeyGen(curve)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.NewPrivateMDToken()
	}

}

func BenchmarkGenTokenPublic(b *testing.B) {

	curve := elliptic.P256()
	pk, _ := KeyGen(curve)

	metadata := make([]byte, 100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.NewPublicMDToken(metadata)
	}
}

func BenchmarkUnblind(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)
	_, uT, u, _ := pk.NewPublicMDToken(metadata)

	// Server: sign blinded token
	uW, proof := sk.PublicMDSign(uT, metadata)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.PublicMDUnblind(uW, u, metadata, proof)
	}

}

func BenchmarkSignPublicMetadata(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)

	_, uT, _, err := pk.NewPublicMDToken(metadata)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.PublicMDSign(uT, metadata)
	}

}

func BenchmarkRedeemPublicMetadata(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	metadata := make([]byte, 100)

	// Client: generate and store (token, bF, bP)
	token, uT, u, err := pk.NewPublicMDToken(metadata)
	if err != nil {
		b.Fatal(err)
	}

	uW, proof := sk.PublicMDSign(uT, metadata)

	W, err := pk.PublicMDUnblind(uW, u, metadata, proof)

	if err != nil {
		b.Fatal(err)
	}

	signed := &SignedToken{token, W, nil, nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.PublicMDRedeem(signed, metadata)
	}
}

func BenchmarkSignPrivateMetadataBit(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	_, uT, _, _, err := pk.NewPrivateMDToken()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.PrivateMDSign(uT, true)
	}

}

func BenchmarkRedeemPrivateMetadataBit(b *testing.B) {

	curve := elliptic.P256()
	pk, sk := KeyGen(curve)

	token, uT, u, v, err := pk.NewPrivateMDToken()
	if err != nil {
		b.Fatal(err)
	}

	uW := sk.PrivateMDSign(uT, true)

	W := pk.PrivateMDUnblind(uW, u, v)

	signed := &SignedToken{token, W, nil, nil}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.PrivateMDRedeem(signed)
	}
}
