package token

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

	"github.com/sachaservan/adveil/ec"
)

type PublicKey struct {
	EC *ec.EC    // elliptic curve
	Pk *ec.Point // token signing key
}

type SecretKey struct {
	EC *ec.EC
	Pk *PublicKey
	Sk *big.Int
}

func KeyGen(curve elliptic.Curve) (*PublicKey, *SecretKey, error) {

	c := &ec.EC{Curve: curve}

	// generate a new elliptic curve point
	k, x, y, _ := elliptic.GenerateKey(curve, rand.Reader)
	X, err := ec.NewPointOnCurve(curve, x, y)
	if err != nil {
		return nil, nil, err
	}

	// secret signing (and verification) key
	ssk := new(big.Int).SetBytes(k)

	pk := &PublicKey{EC: c, Pk: X}
	sk := &SecretKey{EC: c, Pk: pk, Sk: ssk}

	return pk, sk, nil
}
