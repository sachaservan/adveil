package anns

import (
	"github.com/ncw/gmp"
	"github.com/sachaservan/vec"
)

// EuclideanLSH is a set of GaussianHashes (which hashes according to Euclidean distance)
type EuclideanLSH struct {
	Hset []Hash
}

// LSH is a set of locality sensitive hash functions
type LSH struct {
	Hset []Hash
}

// NewEuclideanLSH samples an LSH for L2 norm with dimension dim and parameters:
// r: width parameter for LSH sampled from p-stable distributions
// k: number of concatenated hash functions for amplification
// see Datar et al. Locality-Sensitive Hashing Scheme Based on p-Stable Distributions
// https://dl.acm.org/doi/pdf/10.1145/997817.997857
// for more details on the construction
func NewEuclideanLSH(dim int, r float64, k int) *LSH {

	hashes := make([]Hash, k)
	for i := range hashes {
		hashes[i] = NewGaussianHash(dim, r)
	}

	return &LSH{
		Hset: hashes,
	}
}

// NewHammingLSH sample an LSH for Hamming distance with paramters:
// dim: dimensionality of vectors
// k: number of concatenated hash functions for amplification
func NewHammingLSH(dim int, k int) *LSH {

	hashes := make([]Hash, k)
	for i := range hashes {
		hashes[i] = NewHammingHash(dim)
	}

	return &LSH{
		Hset: hashes,
	}
}

// GetHashSet returns the set of hashes comprising the LSH
func (lsh *LSH) GetHashSet() []Hash {
	return lsh.Hset
}

// EncodeHashesToInt returns an encoding (single gmp.Int) of all the hashes
func EncodeHashesToInt(values ...*gmp.Int) *gmp.Int {

	// NOTE: this encoding strategy assumes that each LSH in Hset is a "1-bit" LSH.
	// To encode concatenation, output SUM 2^i * h_i(x) which encodes the bits as an integer

	// TODO: make this encoding compatible with any base. Currently slow and inefficient.

	res := gmp.NewInt(0)
	pow := gmp.NewInt(2)
	for _, d := range values {
		d.Mul(d, pow)
		res.Add(res, d)

		// next power of 2
		pow.Mul(pow, gmp.NewInt(2))
	}

	return res
}

// Digest outputs the encoded LSH digest of input v
func (lsh *LSH) Digest(v *vec.Vec) *gmp.Int {

	digests := make([]*gmp.Int, len(lsh.Hset))
	for i, h := range lsh.Hset {
		d := h.Digest(v)
		digests[i] = d
	}

	return EncodeHashesToInt(digests...)
}

// StringDigest outputs a string representation of the digest of v
func (lsh *LSH) StringDigest(v *vec.Vec) string {
	res := lsh.Digest(v)
	return string(res.Bytes())
}
