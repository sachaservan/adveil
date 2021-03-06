package crypto

import (
	"crypto/elliptic"
	crand "crypto/rand"
	"math/big"
)

type PublicKey struct {
	Pk *Point
}

type SecretKey struct {
	Sk *big.Int
}

func KeyGen(curve elliptic.Curve) (*PublicKey, *SecretKey) {
	k, x, y, _ := elliptic.GenerateKey(curve, crand.Reader)
	p := &Point{curve, x, y}
	sk := new(big.Int).SetBytes(k)
	return &PublicKey{Pk: p}, &SecretKey{Sk: sk}
}
