// Modified from: https://github.com/privacypass/challenge-bypass-server/blob/master/LICENSE

package ec

import (
	"crypto"
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"
)

var (
	ErrIncompatibleCurveParams = errors.New("incompatible curve")
)

type h2cMethod string

const (
	INC_ITER = 20
	H2C_SWU  = h2cMethod("swu")
	H2C_INC  = h2cMethod("increment")
)

type H2CObject interface {
	HashToCurve(data []byte) (*Point, error)
	Curve() elliptic.Curve
	Hash() crypto.Hash
	Method() string
}

type CurveParams struct {
	Curve  string `json:"curve"`
	Hash   string `json:"hash"`
	Method string `json:"method"`
}

func GetDefaultCurveHash() (H2CObject, error) {
	curveParams := &CurveParams{Curve: "p256", Hash: "sha256", Method: "increment"}
	h2cObj, err := curveParams.GetH2CObj()
	return h2cObj, err
}

// GetH2CObj parses a map of curve parameters for the correct settings
func (curveParams *CurveParams) GetH2CObj() (H2CObject, error) {
	switch curveParams.Curve {
	case "p256":
		params := &h2c{
			curve: elliptic.P256(),
			hash:  crypto.SHA256,
			seed:  []byte("some point generation seed"),
		}

		return &P256SHA256SWU{params}, nil
	}
	return nil, fmt.Errorf("%s, curve: %v, hash: %v, method: %s",
		ErrIncompatibleCurveParams.Error(),
		curveParams.Curve, curveParams.Hash, curveParams.Method)
}

type h2c struct {
	curve elliptic.Curve
	hash  crypto.Hash
	seed  []byte
}

func (obj *h2c) Curve() elliptic.Curve { return obj.curve }
func (obj *h2c) Hash() crypto.Hash     { return obj.hash }

// P256SHA256SWU calculates the Simplified SWU encoding by Brier et al.
// given in "Efficient Indifferentiable Hashing into Ordinary Elliptic Curves".
// It assumes that curve is one of the NIST curves; thus a=-3 and p=3 mod 4.
type P256SHA256SWU struct{ *h2c }

func (obj *P256SHA256SWU) Method() string { return string(H2C_SWU) }

func (obj *P256SHA256SWU) HashToCurve(data []byte) (*Point, error) {
	if obj.curve != elliptic.P256() || obj.hash != crypto.SHA256 {
		return nil, fmt.Errorf("%s for P256SHA256SWU, curve: %v, hash: %v, method %s",
			ErrIncompatibleCurveParams.Error(), obj.curve,
			obj.hash, obj.Method())
	}
	// Compute hash-to-curve based on the contents of the "method" field
	t, err := obj.hashToBaseField(data)
	if err != nil {
		return nil, err
	}
	P, err := obj.simplifiedSWU(t)
	if err != nil {
		return nil, err
	}

	return P, nil
}

// Hashes bytes to a big.Int that will be interpreted as a field element
func (obj *P256SHA256SWU) hashToBaseField(data []byte) (*big.Int, error) {
	byteLen := getFieldByteLength(obj.curve)
	h := obj.hash.New()
	_, err := h.Write(obj.seed)
	if err != nil {
		return nil, err
	}
	_, err = h.Write(data)
	if err != nil {
		return nil, err
	}
	sum := h.Sum(nil)
	t := new(big.Int).SetBytes(sum[:byteLen])
	t.Mod(t, obj.curve.Params().P)
	return t, nil
}

func (obj *P256SHA256SWU) simplifiedSWU(t *big.Int) (*Point, error) {
	var u, t0, y2, bDivA, g, pPlus1Div4, x, y big.Int
	e := obj.curve.Params()
	p := e.P
	A := big.NewInt(-3)
	B := e.B
	// bDivA = -B/A
	bDivA.ModInverse(A, p)
	bDivA.Mul(&bDivA, B)
	bDivA.Neg(&bDivA)
	bDivA.Mod(&bDivA, p)
	// pplus1div4 = (p+1)/4
	pPlus1Div4.SetInt64(1)
	pPlus1Div4.Add(&pPlus1Div4, p)
	pPlus1Div4.Rsh(&pPlus1Div4, 2)
	// u = -t^2
	u.Mul(t, t)
	u.Neg(&u)
	u.Mod(&u, p)
	// t0 = 1/(u^2+u)
	t0.Mul(&u, &u)
	t0.Add(&t0, &u)
	t0.Mod(&t0, p)
	// if t is {0,1,-1} returns error (point at infinity)
	if t0.Sign() == 0 {
		return nil, ErrPointOffCurve
	}
	t0.ModInverse(&t0, p)
	// x = (-B/A)*( 1+1/(u^2+u) ) = bDivA*(1+t0)
	x.SetInt64(1)
	x.Add(&x, &t0)
	x.Mul(&x, &bDivA)
	x.Mod(&x, p)
	// g = (x^2+A)*x+B
	g.Mul(&x, &x)
	g.Mod(&g, p)
	g.Add(&g, A)
	g.Mul(&g, &x)
	g.Mod(&g, p)
	g.Add(&g, B)
	g.Mod(&g, p)
	// y = g^((p+1)/4)
	y.Exp(&g, &pPlus1Div4, p)
	// if y^2 != g, then x = -t^2*x and y = (-1)^{(p+1)/4}*t^3*y
	y2.Mul(&y, &y)
	y2.Mod(&y2, p)
	if y2.Cmp(&g) != 0 {
		// x = -t^2*x
		x.Mul(&x, &u)
		x.Mod(&x, p)
		// y = t^3*y
		y.Mul(&y, &u)
		y.Mod(&y, p)
		y.Neg(&y)
		y.Mul(&y, t)
		y.Mod(&y, p)
	}
	return NewPointOnCurve(obj.curve, &x, &y)
}

func getFieldByteLength(curve elliptic.Curve) int {
	return (curve.Params().BitSize + 7) >> 3
}
