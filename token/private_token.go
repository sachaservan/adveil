// Implementation of PMBTokens
// With 1-bit of private metadata: Construction 4 of https://eprint.iacr.org/2020/072.pdf
// With n-bits of public metadata: https://eprint.iacr.org/2021/203.pdf
package token

import (
	crand "crypto/rand"
	"math/big"

	"github.com/sachaservan/adveil/crypto"
)

func (pk *PublicKey) NewToken() ([]byte, *crypto.Point, []byte, []byte, error) {

	t, _, err := crypto.NewRandomPoint()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// T = H(t)
	T, err := h2cObj.HashToCurve(t)
	if err != nil {
		panic(err)
	}

	// P := u^-1(T - vG)
	P, u, v := Blind(T)
	return t, P, u, v, nil
}

// Blind generates a multiplicative and additive blinding values (u, v),
// and returns (u^-1)(P - vG) along with the random values (u,v)
func Blind(P *crypto.Point) (*crypto.Point, []byte, []byte) {
	u, _, err := crypto.RandomCurveScalar(P.Curve, crand.Reader)
	if err != nil {
		panic(err)
	}
	v, _, err := crypto.RandomCurveScalar(P.Curve, crand.Reader)
	if err != nil {
		panic(err)
	}

	uInv := new(big.Int).SetBytes(u)
	uInv.ModInverse(uInv, P.Curve.Params().N) // u^-1
	x, y := P.Curve.ScalarBaseMult(v)

	neg := new(big.Int).Sub(P.Curve.Params().P, y) // -vG
	x, y = P.Curve.Add(P.X, P.Y, x, neg)           // (P - vG)
	x, y = P.Curve.ScalarMult(x, y, uInv.Bytes())  // u^-1(P - vG)
	B := &crypto.Point{Curve: P.Curve, X: x, Y: y}

	return B, u, v
}

// Sign (computed by the verifier) signs a blinded token.
// The signature is valid or invalid based on the bit b.
func (sk *SecretKey) Sign(B *crypto.Point, bit bool) *SignedBlindToken {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// generate okamoto-schnorr randomization factor
	s, _, err := crypto.NewRandomPoint()
	if err != nil {
		panic(err)
	}

	s = append(s, B.X.Bytes()...)
	s = append(s, B.Y.Bytes()...)
	S, err := h2cObj.HashToCurve(s) // S = H(s||B.x||B.y)
	if err != nil {
		panic(err)
	}

	// sign the token and randomize
	skr := sk.Skr.Bytes() // use the randomization key to randomize the signature
	x, y := S.Curve.ScalarMult(S.X, S.Y, skr)
	S = &crypto.Point{Curve: S.Curve, X: x, Y: y} // yS

	sks := sk.Sks.Bytes() // use the signing key to sign the token
	if !bit {
		// generate a garbage token by signing with a random key
		var err error
		sks, _, err = crypto.RandomCurveScalar(B.Curve, crand.Reader)
		if err != nil {
			panic(err)
		}
	}

	x, y = B.Curve.ScalarMult(B.X, B.Y, sks)
	W := &crypto.Point{Curve: B.Curve, X: x, Y: y} // xB (signed token)

	x, y = W.Curve.Add(W.X, W.Y, S.X, S.Y)
	W = &crypto.Point{Curve: B.Curve, X: x, Y: y} // xB + yS (randomized signed token)

	return &SignedBlindToken{B: B, W: W, S: s}
}

// Unblind (computed by the prover) removes the given blinding
// factor from the signed token.
func (pk *PublicKey) Unblind(T *SignedBlindToken, t []byte, u []byte, v []byte) *SignedToken {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	W := T.W
	B := T.B

	// replicate the okamoto-schnorr randomization factor
	s := T.S
	s = append(s, B.X.Bytes()...)
	s = append(s, B.Y.Bytes()...)
	S, err := h2cObj.HashToCurve(s) // H(s||B.x||B.y)
	if err != nil {
		panic(err)
	}

	x, y := W.Curve.ScalarMult(S.X, S.Y, u)
	S = &crypto.Point{Curve: S.Curve, X: x, Y: y} // uS

	x, y = W.Curve.ScalarMult(pk.H.X, pk.H.Y, v) // vH
	x, y = W.Curve.Add(S.X, S.Y, x, y)
	S = &crypto.Point{Curve: S.Curve, X: x, Y: y} // uS + vH

	x, y = W.Curve.Add(pk.Pks.X, pk.Pks.Y, pk.Pkr.X, pk.Pkr.Y)
	X := &crypto.Point{Curve: W.Curve, X: x, Y: y} // Pkr + Pks = xG + yH

	x, y = W.Curve.ScalarMult(W.X, W.Y, u)    // uW
	x1, y1 := W.Curve.ScalarMult(X.X, X.Y, v) // vX
	x, y = W.Curve.Add(x, y, x1, y1)
	W = &crypto.Point{Curve: W.Curve, X: x, Y: y} // uW + vX
	return &SignedToken{T: t, S: S, W: W}
}

func (sk *SecretKey) Redeem(T *SignedToken) bool {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	P, err := h2cObj.HashToCurve(T.T)
	if err != nil {
		panic(err)
	}

	sks := sk.Sks.Bytes()
	x, y := P.Curve.ScalarMult(P.X, P.Y, sks)
	P = &crypto.Point{Curve: P.Curve, X: x, Y: y}

	S := T.S
	skr := sk.Skr.Bytes()
	x, y = P.Curve.ScalarMult(S.X, S.Y, skr)
	S = &crypto.Point{Curve: P.Curve, X: x, Y: y}

	x, y = S.Curve.Add(P.X, P.Y, S.X, S.Y)
	Z := &crypto.Point{Curve: S.Curve, X: x, Y: y}

	// are points equal?
	return Z.X.Cmp(T.W.X) == 0 && Z.Y.Cmp(T.W.Y) == 0
}
