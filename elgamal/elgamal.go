package elgamal

import (
	"adveil/crypto"
	crand "crypto/rand"
	_ "crypto/sha256"
	"math/big"
)

type Ciphertext struct {
	A *crypto.Point // m + sk*r*G
	B *crypto.Point // r*G
}

func (pk *PublicKey) Encrypt(m *crypto.Point) *Ciphertext {
	r, _ := crand.Int(crand.Reader, m.Curve.Params().N)
	x, y := m.Curve.ScalarBaseMult(r.Bytes())
	B := &crypto.Point{Curve: m.Curve, X: x, Y: y}

	x, y = m.Curve.ScalarMult(pk.Pk.X, pk.Pk.Y, r.Bytes())
	x, y = m.Curve.Add(m.X, m.Y, x, y)
	A := &crypto.Point{Curve: m.Curve, X: x, Y: y}

	return &Ciphertext{A, B}
}

func (sk *SecretKey) Decrypt(ct *Ciphertext) *crypto.Point {
	curve := ct.A.Curve
	x, y := curve.ScalarMult(ct.B.X, ct.B.Y, sk.Sk.Bytes())
	x, y = curve.Add(ct.A.X, ct.A.Y, x, new(big.Int).Sub(curve.Params().P, y))

	return &crypto.Point{curve, x, y}
}

func (sk *SecretKey) PartialDecrypt(ct *Ciphertext) *Ciphertext {
	curve := ct.A.Curve
	x, y := curve.ScalarMult(ct.B.X, ct.B.Y, sk.Sk.Bytes())
	B := &crypto.Point{curve, x, y}

	return &Ciphertext{ct.A, B}
}

func (pk *PublicKey) RandomPoint() *crypto.Point {

	_, p, err := crypto.NewRandomPoint()
	if err != nil {
		panic(err)
	}

	return p
}
