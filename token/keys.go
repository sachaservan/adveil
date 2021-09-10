package token

import (
	"crypto/elliptic"
	crand "crypto/rand"
	"math/big"

	"github.com/sachaservan/adveil/crypto"
)

type PublicKey struct {
	Pks *crypto.Point // signing key
	Pkr *crypto.Point // randomization key
	H   *crypto.Point
}

type SecretKey struct {
	Pks *crypto.Point
	Pkr *crypto.Point
	Sks *big.Int
	Skr *big.Int
}

func KeyGen(curve elliptic.Curve) (*PublicKey, *SecretKey) {
	k, x, y, _ := elliptic.GenerateKey(curve, crand.Reader)
	Ps := &crypto.Point{curve, x, y}
	sks := new(big.Int).SetBytes(k)

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}
	H, err := h2cObj.HashToCurve(append(x.Bytes(), y.Bytes()...))
	if err != nil {
		panic(err)
	}

	k, x, y, _ = elliptic.GenerateKey(curve, crand.Reader)
	x, y = curve.ScalarMult(H.X, H.Y, k)
	Pr := &crypto.Point{curve, x, y}
	skr := new(big.Int).SetBytes(k)

	pk := &PublicKey{Pks: Ps, Pkr: Pr, H: H}
	sk := &SecretKey{Pks: Ps, Pkr: Pr, Sks: sks, Skr: skr}
	return pk, sk
}
