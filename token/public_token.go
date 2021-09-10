package token

import (
	"bytes"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha256"
	"errors"
	"math/big"

	"github.com/sachaservan/adveil/crypto"
)

// NewPublicMDToken generates a token t, blinded curve point H(t)^{1/u} and blinding factor u.
// Returns (t, B, u)
func (pk *PublicKey) NewPublicMDToken() ([]byte, *crypto.Point, []byte, error) {

	t, _, err := crypto.NewRandomPoint()
	if err != nil {
		return nil, nil, nil, err
	}

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// compute H(t)
	T, err := h2cObj.HashToCurve(t)
	if err != nil {
		return nil, nil, nil, err
	}

	// T := u^-1(T)
	u, _, err := crypto.RandomCurveScalar(T.Curve, crand.Reader)
	if err != nil {
		panic(err)
	}

	uInv := new(big.Int).SetBytes(u)
	uInv.ModInverse(uInv, T.Curve.Params().N) // u^-1

	x, y := T.Curve.ScalarMult(T.X, T.Y, uInv.Bytes()) // (u^-1)T
	B := &crypto.Point{Curve: T.Curve, X: x, Y: y}

	return t, B, u, nil
}

// PublicMDSign (computed by the verifier) signs a blinded token
// and compute a DLEQ proof attesting to the validity of the signature
func (sk *SecretKey) PublicMDSign(B *crypto.Point, md []byte) *SignedBlindTokenWithMD {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// compute H_3(md)
	d := new(big.Int).SetBytes(h3(md))
	d.Mod(d, B.Curve.Params().N) // interpret as field element
	d.Add(d, sk.Sks)             // compute sk + d

	e := new(big.Int)
	e.ModInverse(d, B.Curve.Params().N) // d^-1

	x, y := sk.Pks.Curve.ScalarBaseMult(e.Bytes())
	U := &crypto.Point{Curve: B.Curve, X: x, Y: y} // (g^pk)^(1/d)

	x, y = B.Curve.ScalarMult(B.X, B.Y, e.Bytes())
	W := &crypto.Point{Curve: B.Curve, X: x, Y: y} // B^(1/d)

	G := &crypto.Point{Curve: sk.Pks.Curve, X: B.Curve.Params().Gx, Y: B.Curve.Params().Gy}
	proof, err := crypto.NewProof(h2cObj.Hash(), G, U, B, W, e) // TODO: use a different hash function for the proof!

	if err != nil {
		panic(err)
	}

	return &SignedBlindTokenWithMD{Curve: B.Curve, W: W, MD: md, Proof: proof}
}

// PublicMDUnblind (computed by the prover) removes the given blinding
// factor from the signed token
func (pk *PublicKey) PublicMDUnblind(T *SignedBlindTokenWithMD, t []byte, u []byte) (*SignedTokenWithMD, error) {

	// Verify new proof
	h2cObj, _ := crypto.GetDefaultCurveHash()
	if !T.Proof.Verify(h2cObj) {
		return nil, errors.New("proof invalid")
	}

	x, y := T.Curve.ScalarMult(T.W.X, T.W.Y, u)
	hashdata := make([]byte, 0)
	hashdata = append(hashdata, T.MD...)
	hashdata = append(hashdata, t...)
	hashdata = append(hashdata, x.Bytes()...)
	hashdata = append(hashdata, y.Bytes()...)
	z := h2(hashdata)

	return &SignedTokenWithMD{Curve: T.Curve, T: t, Z: z, MD: T.MD}, nil
}

func (sk *SecretKey) PublicMDRedeem(T *SignedTokenWithMD, md []byte) bool {

	h2cObj, err := crypto.GetDefaultCurveHash()
	if err != nil {
		panic(err)
	}

	// compute H_3(md)
	d := new(big.Int).SetBytes(h3(md))
	d.Mod(d, T.Curve.Params().N) // interpret as field element
	d.Add(d, sk.Sks)             // compute sk + d

	e := new(big.Int)
	e.ModInverse(d, T.Curve.Params().N) // d^-1

	// compute H(t)
	P, err := h2cObj.HashToCurve(T.T)
	if err != nil {
		panic(err)
	}

	x, y := P.Curve.ScalarMult(P.X, P.Y, e.Bytes())
	hashdata := make([]byte, 0)
	hashdata = append(hashdata, md...)
	hashdata = append(hashdata, T.T...)
	hashdata = append(hashdata, x.Bytes()...)
	hashdata = append(hashdata, y.Bytes()...)
	z := h2(hashdata)

	// is the result the same?
	return bytes.Equal(T.Z, z)
}

func h2(data []byte) []byte {
	// compute H_3 = SHA(3)
	// TODO: find a less hacky way to do this
	hkey := make([]byte, 0)
	hkey = append(hkey, byte('2'))
	h := hmac.New(sha256.New, hkey)

	// write data to the hash
	h.Write([]byte(data))
	return h.Sum(nil)
}

func h3(data []byte) []byte {
	// compute H_3 = SHA(3)
	// TODO: find a less hacky way to do this
	hkey := make([]byte, 0)
	hkey = append(hkey, byte('3'))
	h := hmac.New(sha256.New, hkey)

	// write data to the hash
	h.Write([]byte(data))
	return h.Sum(nil)
}
