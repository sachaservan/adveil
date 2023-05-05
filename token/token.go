package token

import (
	"crypto/elliptic"
	"crypto/rand"
	"math/big"

	"github.com/sachaservan/adveil/ec"
)

type BlindToken struct {
	Curve elliptic.Curve
	T     []byte    // token value t
	B     *ec.Point // (blind) token
	U     *big.Int  // blinding factor
	V     *big.Int  // randomization value
}

type SignedBlindToken struct {
	Curve elliptic.Curve
	W     *ec.Point // (blind) signature
}

type SignedToken struct {
	Curve elliptic.Curve
	T     []byte    // token value t
	S     *ec.Point // (unblind) signature
}

func (pk *PublicKey) NewToken() (*BlindToken, error) {

	t := make([]byte, 16)
	_, err := rand.Read(t)
	if err != nil {
		return nil, err
	}

	h2cObj, err := ec.GetDefaultCurveHash()
	if err != nil {
		return nil, err
	}

	// P = H(t)
	P, err := h2cObj.HashToCurve(t)
	if err != nil {
		return nil, err
	}

	// B := u^-1(P - vG)
	B, u, v := pk.Blind(P)
	return &BlindToken{T: t, B: B, U: u, V: v}, nil
}

// Blind generates a multiplicative and additive blinding values (u, v),
// and returns (u^-1)(P - vG) along with the random values (u,v)
func (pk *PublicKey) Blind(P *ec.Point) (*ec.Point, *big.Int, *big.Int) {

	_, u, err := pk.EC.RandomCurveScalar(rand.Reader)
	if err != nil {
		panic(err)
	}

	_, v, err := pk.EC.RandomCurveScalar(rand.Reader)
	if err != nil {
		panic(err)
	}

	uInv := new(big.Int).SetBytes(u.Bytes())
	uInv.ModInverse(uInv, P.Curve.Params().N) // u^-1

	vG := pk.EC.ScalarBaseMult(v) // vG
	vG = pk.EC.Inverse(vG)        // -vG

	B := pk.EC.Add(P, vG)         // (P - vG)
	B = pk.EC.ScalarMult(B, uInv) // u^-1(P - vG)

	return B, u, v
}

// Sign (computed by the verifier) signs a blinded token.
// The signature is valid or invalid based on the bit b.
func (sk *SecretKey) Sign(B *ec.Point) (*SignedBlindToken, error) {

	pk := sk.Pk

	// use the signing key to sign the token
	xB := pk.EC.ScalarMult(B, sk.Sk) // xB (x = sk)

	return &SignedBlindToken{W: xB}, nil
}

// Unblind (computed by the prover) removes the given blinding
// factor from the signed token.
func (pk *PublicKey) Unblind(sbt *SignedBlindToken, bt *BlindToken) *SignedToken {

	W := sbt.W // xB

	uW := pk.EC.ScalarMult(W, bt.U)     // uW = uxB
	vX := pk.EC.ScalarMult(pk.Pk, bt.V) // vX = vxG

	S := pk.EC.Add(uW, vX) // uW + vX = vxG + xP - xvG = xP

	return &SignedToken{T: bt.T, S: S}
}

func (sk *SecretKey) Redeem(T *SignedToken) (bool, error) {

	pk := sk.Pk

	h2cObj, err := ec.GetDefaultCurveHash()
	if err != nil {
		return false, err
	}

	// P = H(t)
	P, err := h2cObj.HashToCurve(T.T)
	if err != nil {
		return false, err
	}

	xP := pk.EC.ScalarMult(P, sk.Sk) // xP

	// are points equal?
	return xP.X.Cmp(T.S.X) == 0 && xP.Y.Cmp(T.S.Y) == 0, nil
}
