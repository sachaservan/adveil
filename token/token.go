// An implementation of an oblivious VRF due to Jarecki et al.
package token

import (
	"adveil/crypto"
	crand "crypto/rand"
	"math/big"
)

type SignedToken struct {
	T []byte        // token bytes
	S *crypto.Point // signature
}

func (pk *PublicKey) NewToken() ([]byte, *crypto.Point, []byte, error) {

	token, T, err := crypto.NewRandomPoint()
	if err != nil {
		return nil, nil, nil, err
	}

	// T = H(token) => Point
	// P := rT
	P, r := Blind(T)
	return token, P, r, nil
}

func (sk *SecretKey) Sign(p *crypto.Point) *crypto.Point {
	curve := p.Curve
	x, y := curve.ScalarMult(p.X, p.Y, sk.Sk.Bytes())
	q := &crypto.Point{Curve: curve, X: x, Y: y}
	return q
}

func (sk *SecretKey) Redeem(token *SignedToken) bool {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	T, err := h2cObj.HashToCurve(token.T)
	if err != nil {
		panic(err)
	}

	sigPrime := sk.Sign(T)

	// are points equal?
	return sigPrime.X.Cmp(token.S.X) == 0 && sigPrime.Y.Cmp(token.S.Y) == 0
}

func (sk *SecretKey) RedeemAndProve(pk *PublicKey, token *SignedToken) (bool, *crypto.Proof) {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	T, err := h2cObj.HashToCurve(token.T)
	if err != nil {
		panic(err)
	}

	S := sk.Sign(T)

	g := &crypto.Point{pk.Pk.Curve, pk.Pk.Curve.Params().Gx, pk.Pk.Curve.Params().Gy}
	proof, err := crypto.NewProof(h2cObj.Hash(), g, pk.Pk, T, S, sk.Sk)
	if err != nil {
		panic(err)
	}

	// are points equal?
	ok := S.X.Cmp(token.S.X) == 0 && S.Y.Cmp(token.S.Y) == 0
	return ok, proof
}

// Blind generates a random blinding factor, scalar multiplies it to the
// supplied point, and returns both the new point and the blinding factor.
func Blind(p *crypto.Point) (*crypto.Point, []byte) {
	r, _, err := crypto.RandomCurveScalar(p.Curve, crand.Reader)
	if err != nil {
		return nil, nil
	}
	x, y := p.Curve.ScalarMult(p.X, p.Y, r)
	a := &crypto.Point{Curve: p.Curve, X: x, Y: y}
	return a, r
}

// Unblind removes the given blinding factor from the point.
func Unblind(p *crypto.Point, blind []byte) *crypto.Point {
	r := new(big.Int).SetBytes(blind)
	r.ModInverse(r, p.Curve.Params().N)
	x, y := p.Curve.ScalarMult(p.X, p.Y, r.Bytes())
	return &crypto.Point{Curve: p.Curve, X: x, Y: y}
}
