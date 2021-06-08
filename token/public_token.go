package token

import (
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/sachaservan/adveil/crypto"
)

func (pk *PublicKey) NewPublicMDToken(md []byte) ([]byte, *crypto.Point, []byte, error) {

	token, _, err := crypto.NewRandomPoint()
	if err != nil {
		return nil, nil, nil, err
	}

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// compute H(t||md)
	data := append(token, md...)
	T, err := h2cObj.HashToCurve(data)
	if err != nil {
		return nil, nil, nil, err
	}

	// P := u^-1(T)
	P, u := PublicMDBlind(T, md)
	return token, P, u, nil
}

// PublicMDBlind generates a multiplicative factor u,
// and returns (u^-1)(P) along with the random values (u,v)
func PublicMDBlind(P *crypto.Point, md []byte) (*crypto.Point, []byte) {
	u, _, err := crypto.RandomCurveScalar(P.Curve, crand.Reader)
	if err != nil {
		return nil, nil
	}

	uInv := new(big.Int).SetBytes(u)
	uInv.ModInverse(uInv, P.Curve.Params().N) // u^-1

	x, y := P.Curve.ScalarMult(P.X, P.Y, uInv.Bytes()) // (u^-1)P
	A := &crypto.Point{Curve: P.Curve, X: x, Y: y}

	return A, u
}

// PublicMDSign (computed by the verifier) signs a blinded token
// and compute a DLEQ proof attesting to the validity of the signature
func (sk *SecretKey) PublicMDSign(P *crypto.Point, md []byte) (*crypto.Point, *crypto.Proof) {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// compute H(md)
	h := hmac.New(sha256.New, []byte(md))
	// Write Data to it
	h.Write([]byte(md))
	d := new(big.Int).SetBytes(h.Sum(nil))
	d.Mod(d, P.Curve.Params().N) // interpret as field element

	d = sk.Sk // TODO: should be (d + sk) but not sure how to add in the EC field yet;

	e := new(big.Int)
	e.ModInverse(d, P.Curve.Params().N) // e^-1

	x, y := sk.Pk.Curve.ScalarBaseMult(e.Bytes())
	U := &crypto.Point{Curve: P.Curve, X: x, Y: y}

	x, y = P.Curve.ScalarMult(P.X, P.Y, e.Bytes())
	W := &crypto.Point{Curve: P.Curve, X: x, Y: y}

	G := &crypto.Point{sk.Pk.Curve, P.Curve.Params().Gx, P.Curve.Params().Gy}
	proof, err := crypto.NewProof(h2cObj.Hash(), G, U, P, W, e)

	if err != nil {
		panic(err)
	}

	return W, proof
}

// PublicMDUnblind (computed by the prover) removes the given blinding
// factor from the signed token
func (pk *PublicKey) PublicMDUnblind(P *crypto.Point, u []byte, md []byte, proof *crypto.Proof) (*crypto.Point, error) {
	// compute H(md)
	h := hmac.New(sha256.New, []byte(md))
	// Write Data to it
	h.Write([]byte(md))
	d := new(big.Int).SetBytes(h.Sum(nil))
	d.Mod(d, P.Curve.Params().N) // interpret as field element

	// TODO:  check to make sure proof is for the correct values
	// U := pk.Pk // TODO: should be (dPk) but then should match PublicMDSign

	// Verify new proof
	h2cObj, _ := crypto.GetDefaultCurveHash()
	if !proof.Verify(h2cObj) {
		return nil, errors.New("proof invalid")
	}

	x, y := P.Curve.ScalarMult(P.X, P.Y, u)
	W := &crypto.Point{Curve: P.Curve, X: x, Y: y}

	return W, nil
}

func (sk *SecretKey) PublicMDRedeem(token *SignedToken, md []byte) bool {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// compute H(t||md)
	data := append(token.T, md...)
	T, err := h2cObj.HashToCurve(data)
	if err != nil {
		return false
	}

	// compute H(md)
	h := hmac.New(sha256.New, []byte(md))
	// Write Data to it
	h.Write([]byte(md))
	d := new(big.Int).SetBytes(h.Sum(nil))
	d.Mod(d, T.Curve.Params().N) // interpret as field element

	d = sk.Sk // TODO: should be (d + sk) but not sure how to add in the EC field yet;

	e := new(big.Int)
	e.ModInverse(d, T.Curve.Params().N) // e^-1

	x, y := T.Curve.ScalarMult(T.X, T.Y, e.Bytes())
	W := &crypto.Point{Curve: T.Curve, X: x, Y: y}

	// are points equal?
	return W.X.Cmp(token.W.X) == 0 && W.Y.Cmp(token.W.Y) == 0
}
