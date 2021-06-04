package anns

import (
	"bytes"
	crand "crypto/rand"
	"encoding/gob"
	"errors"
	"math"
	"math/big"
	"math/rand"

	"github.com/sachaservan/vec"

	"github.com/ncw/gmp"
)

// Hash is an abstract hash function
type Hash interface {
	Digest(*vec.Vec) *gmp.Int
	StringDigest(*vec.Vec) string
}

// UniversalHash is a universal hash function h(x) = r1*x + r2 mod n
type UniversalHash struct {
	r1 *gmp.Int
	r2 *gmp.Int
	n  *gmp.Int
}

// GaussianHash is locality sensitive with respect to L2 distance
type GaussianHash struct {
	a *vec.Vec // fixed-point encoded vector of gaussian random variables
	b float64  // uniformly random value in the range [0, r]
	r float64
}

type universalHashMarshallWrapper struct {
	R1 *gmp.Int
	R2 *gmp.Int
	N  *gmp.Int
}

type gaussianlHashMarshallWrapper struct {
	A *vec.Vec // fixed-point encoded vector of gaussian random variables
	B float64  // uniformly random value in the range [0, r]
	R float64
}

// NewGaussianHash generates a new locality sensitive Gaussian hash for L2 distance metric
func NewGaussianHash(dim int, r float64) *GaussianHash {

	a := make([]float64, dim)

	// generate a random float by sampling a scaled int
	b := float64(rand.Intn(int(r*100000.0)) / 100000.0)

	for i := range a {
		a[i] = rand.NormFloat64()
	}

	return &GaussianHash{
		vec.NewVec(a), b, r,
	}
}

// NewUniversalHash samples a new universal hash with range hashBytes
func NewUniversalHash(hashBytes int) *UniversalHash {
	var r1, r2, n *big.Int
	err := errors.New("")
	for err != nil {
		n, err = crand.Prime(crand.Reader, hashBytes*8)
	}

	err = errors.New("")
	for err != nil {
		r1, err = crand.Int(crand.Reader, n)
	}

	err = errors.New("")
	for err != nil {
		r2, err = crand.Int(crand.Reader, n)
	}

	return &UniversalHash{
		gmp.NewInt(0).SetBytes(r1.Bytes()),
		gmp.NewInt(0).SetBytes(r2.Bytes()),
		gmp.NewInt(0).SetBytes(n.Bytes()),
	}
}

// NewUniversalHashWithModulus samples a new universal hash with range [0...n]
func NewUniversalHashWithModulus(n *gmp.Int) *UniversalHash {
	var r1, r2 *big.Int
	nBig := big.NewInt(0).SetBytes(n.Bytes())

	err := errors.New("")
	for err != nil {
		r1, err = crand.Int(crand.Reader, nBig)
	}

	err = errors.New("")
	for err != nil {
		r2, err = crand.Int(crand.Reader, nBig)
	}

	return &UniversalHash{
		gmp.NewInt(0).SetBytes(r1.Bytes()),
		gmp.NewInt(0).SetBytes(r2.Bytes()),
		n,
	}
}

// GetHashParameters returns the gaussian hash parameters
func (h *GaussianHash) GetHashParameters() (*vec.Vec, float64, float64) {
	return h.a, h.b, h.r
}

// Digest returns the hash of the vector v
func (h *GaussianHash) Digest(v *vec.Vec) *gmp.Int {

	// 1. compute the inner product
	res, _ := h.a.Dot(v)

	// 2. add the constant term and normalize
	res += h.b
	res /= float64(h.r)
	res = math.Abs(res) // make positive to avoid encoding issues

	return gmp.NewInt(int64(math.Floor(res)))
}

// StringDigest returns the evaluation of the hash on v encoded as a string
func (h *GaussianHash) StringDigest(v *vec.Vec) string {
	return string(h.Digest(v).Bytes())
}

// Digest returns the evaluation of the hash on s
func (h *UniversalHash) Digest(s *gmp.Int) *gmp.Int {
	sInt := gmp.NewInt(0)
	sInt.Add(sInt, s).Mod(sInt, h.n)
	sInt.Mul(sInt, h.r1).Add(sInt, h.r2).Mod(sInt, h.n)

	return sInt
}

// BigVecDigest returns the evaluation of the hash on s
func (h *UniversalHash) BigVecDigest(s *vec.BigVec) *vec.BigVec {

	res := make([]*gmp.Int, s.Size())
	for i := range res {
		coordHash := new(gmp.Int).SetBytes(s.Coord(i).Bytes())
		coordHash.Mul(coordHash, h.r1).Add(coordHash, h.r2).Mod(coordHash, h.n)
		res[i] = coordHash
	}

	return vec.NewBigVec(res)
}

// GetHashParameters returns the guassian hash parameters
func (h *UniversalHash) GetHashParameters() (*gmp.Int, *gmp.Int, *gmp.Int) {
	return h.r1, h.r2, h.n
}

// StringDigest returns the evaluation of the hash on s encoded as a string
func (h *UniversalHash) StringDigest(s *gmp.Int) string {
	return string(h.Digest(s).Bytes())
}

// MarshalBinary is needed in order to encode/decode
// ciphertexts type since (for now) ciphertext
// can only be converted to bytes and back
func (h *UniversalHash) MarshalBinary() ([]byte, error) {

	// wrap struct
	w := universalHashMarshallWrapper{
		h.r1,
		h.r2,
		h.n,
	}

	// use default gob encoder
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(w); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary is needed in order to encode/decode
// ciphertexts type since (for now) ciphertext
// can only be converted to bytes and back
func (h *UniversalHash) UnmarshalBinary(data []byte) error {

	if len(data) == 0 {
		return nil
	}

	w := universalHashMarshallWrapper{}

	reader := bytes.NewReader(data)
	dec := gob.NewDecoder(reader)
	if err := dec.Decode(&w); err != nil {
		return err
	}

	h.r1 = w.R1
	h.r2 = w.R2
	h.n = w.N

	return nil
}

// MarshalBinary is needed in order to encode/decode
// ciphertexts type since (for now) ciphertext
// can only be converted to bytes and back
func (h *GaussianHash) MarshalBinary() ([]byte, error) {

	// wrap struct
	w := gaussianlHashMarshallWrapper{
		h.a,
		h.b,
		h.r,
	}

	// use default gob encoder
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(w); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnmarshalBinary is needed in order to encode/decode
// ciphertexts type since (for now) ciphertext
// can only be converted to bytes and back
func (h *GaussianHash) UnmarshalBinary(data []byte) error {

	if len(data) == 0 {
		return nil
	}

	w := gaussianlHashMarshallWrapper{}

	reader := bytes.NewReader(data)
	dec := gob.NewDecoder(reader)
	if err := dec.Decode(&w); err != nil {
		return err
	}

	h.a = w.A
	h.b = w.B
	h.r = w.R

	return nil
}
