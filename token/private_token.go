// Implementation of PMBTokens
// With 1-bit of private metadata: Construction 4 of https://eprint.iacr.org/2020/072.pdf
// With n-bits of public metadata: https://eprint.iacr.org/2021/203.pdf
package token

import (
	crand "crypto/rand"
	"math/big"

	"github.com/sachaservan/adveil/crypto"
)

func (pk *PublicKey) NewPrivateMDToken() ([]byte, *crypto.Point, []byte, []byte, error) {

	token, T, err := crypto.NewRandomPoint()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// T = H(token) => Point
	// P := u^-1(T - vG)
	P, u, v := PrivateMDBlind(T)
	return token, P, u, v, nil
}

// PrivateMDBlind generates a multiplicative and additive blinding values (u, v),
// and returns (u^-1)(P - vG) along with the random values (u,v)
func PrivateMDBlind(P *crypto.Point) (*crypto.Point, []byte, []byte) {
	u, _, err := crypto.RandomCurveScalar(P.Curve, crand.Reader)
	if err != nil {
		return nil, nil, nil
	}
	v, _, err := crypto.RandomCurveScalar(P.Curve, crand.Reader)
	if err != nil {
		return nil, nil, nil
	}

	uInv := new(big.Int).SetBytes(u)
	uInv.ModInverse(uInv, P.Curve.Params().N) // u^-1
	x, y := P.Curve.ScalarBaseMult(v)

	neg := new(big.Int).Sub(P.Curve.Params().P, y) // -vG
	x, y = P.Curve.Add(P.X, P.Y, x, neg)           // (P - vG)
	x, y = P.Curve.ScalarMult(x, y, uInv.Bytes())  // u^-1(P - vG)
	A := &crypto.Point{Curve: P.Curve, X: x, Y: y}

	return A, u, v
}

// PrivateMDSign (computed by the verifier) signs a blinded token with private
// metadata bit b (results in either valid or invalid token based on b)
func (sk *SecretKey) PrivateMDSign(P *crypto.Point, bit bool) *crypto.Point {

	scal := sk.Sk.Bytes()
	if !bit {
		// generate a garbage token by signing with a random key
		var err error
		scal, _, err = crypto.RandomCurveScalar(P.Curve, crand.Reader)
		if err != nil {
			return nil
		}
	}

	x, y := P.Curve.ScalarMult(P.X, P.Y, scal)
	W := &crypto.Point{Curve: P.Curve, X: x, Y: y}
	return W
}

// PrivateMDUnblind (computed by the prover) removes the given blinding
// factor from the signed token
func (pk *PublicKey) PrivateMDUnblind(P *crypto.Point, u []byte, v []byte) *crypto.Point {
	x, y := P.Curve.ScalarMult(P.X, P.Y, u)
	x1, y1 := P.Curve.ScalarMult(pk.Pk.X, pk.Pk.Y, v)
	x, y = P.Curve.Add(x, y, x1, y1)
	return &crypto.Point{Curve: P.Curve, X: x, Y: y}
}

func (sk *SecretKey) PrivateMDRedeem(token *SignedToken) bool {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	T, err := h2cObj.HashToCurve(token.T)
	if err != nil {
		panic(err)
	}

	sigPrime := sk.PrivateMDSign(T, true)

	// are points equal?
	return sigPrime.X.Cmp(token.W.X) == 0 && sigPrime.Y.Cmp(token.W.Y) == 0
}
