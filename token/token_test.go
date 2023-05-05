package token

import (
	"bytes"
	"crypto/elliptic"
	_ "crypto/sha256"
	"encoding/json"
	"testing"
)

func TestTokenProtocol(t *testing.T) {

	curve := elliptic.P256()
	pk, sk, _ := KeyGen(curve)

	// Client: generate and store (token, bF, bP)
	bt, err := pk.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	// Server: sign blinded token
	sbt, _ := sk.Sign(bt.B)

	// Client: unblind signature
	W := pk.Unblind(sbt, bt)

	// Server: redeem unblinded token and signature
	valid, _ := sk.Redeem(W)
	if !valid {
		t.Fatal("failed redemption")
	}
}

func BenchmarkGenToken(b *testing.B) {

	curve := elliptic.P256()
	pk, _, _ := KeyGen(curve)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.NewToken()
	}
}

func BenchmarkTokenUnblind(b *testing.B) {

	curve := elliptic.P256()
	pk, sk, _ := KeyGen(curve)

	bt, err := pk.NewToken()
	if err != nil {
		b.Fatal(err)
	}

	sbt, _ := sk.Sign(bt.B)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pk.Unblind(sbt, bt)
	}

}

func BenchmarkTokenSign(b *testing.B) {

	curve := elliptic.P256()
	pk, sk, _ := KeyGen(curve)

	bt, err := pk.NewToken()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Sign(bt.B)
	}

}

func BenchmarkTokenRedeem(b *testing.B) {

	curve := elliptic.P256()
	pk, sk, _ := KeyGen(curve)

	bt, err := pk.NewToken()
	if err != nil {
		b.Fatal(err)
	}

	sbt, _ := sk.Sign(bt.B)

	W := pk.Unblind(sbt, bt)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sk.Redeem(W)
	}
}

func TestMarshall(t *testing.T) {

	curve := elliptic.P256()
	pk, _, _ := KeyGen(curve)

	bt, err := pk.NewToken()
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(bt)

	if err != nil {
		t.Fatal(err)
	}

	R := &BlindToken{}
	err = json.Unmarshal(data, R)

	if err != nil {
		t.Fatal(err)
	}

	if !pk.EC.IsEqual(bt.B, R.B) || bt.U.Cmp(R.U) != 0 || bt.V.Cmp(R.V) != 0 || bytes.Compare(bt.T, R.T) != 0 {
		t.Fatalf("recovered point is not valid")
	}

}
