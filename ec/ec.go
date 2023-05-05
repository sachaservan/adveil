// Modified from: https://github.com/privacypass/challenge-bypass-server/blob/master/LICENSE

package ec

import (
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
)

var (
	ErrInvalidPoint     = errors.New("marshaled point was invalid")
	ErrNoPointFound     = errors.New("hash_to_curve failed to find a point")
	ErrPointOffCurve    = errors.New("point is not on curve")
	ErrUnspecifiedCurve = errors.New("must specify an elliptic curve")
)

type EC struct {
	Curve elliptic.Curve
}

type Point struct {
	Curve elliptic.Curve
	X, Y  *big.Int
}

func (point *Point) Copy() *Point {
	return &Point{
		X: new(big.Int).SetBytes(point.X.Bytes()),
		Y: new(big.Int).SetBytes(point.Y.Bytes())}
}

// This is just a bitmask with the number of ones starting at 8 then
// incrementing by index. To account for fields with bitsizes that are not a whole
// number of bytes, we mask off the unnecessary bits. h/t agl
var mask = []byte{0xff, 0x1, 0x3, 0x7, 0xf, 0x1f, 0x3f, 0x7f}

// GeneratorPoint: the base point of the elliptic curve
func (ec *EC) GeneratorPoint() (*Point, error) {

	s := big.NewInt(1)
	x, y := ec.Curve.ScalarBaseMult(s.Bytes())
	return &Point{ec.Curve, x, y}, nil
}

// IdentityPoint: the point at infinity
func (ec *EC) IdentityPoint() *Point {

	s := big.NewInt(0)
	x, y := ec.Curve.ScalarBaseMult(s.Bytes())
	return &Point{ec.Curve, x, y}
}

// NewPoint: Generates a new point on the curve specified in curveParams.
func (ec *EC) NewPoint(s *big.Int) (*Point, error) {
	x, y := ec.Curve.ScalarBaseMult(s.Bytes())
	return &Point{ec.Curve, x, y}, nil
}

func NewPointOnCurve(curve elliptic.Curve, x, y *big.Int) (*Point, error) {
	return &Point{curve, x, y}, nil
}

// NewRandomPoint: Generates a new random point on the curve specified in curveParams.
func (ec *EC) NewRandomPoint() ([]byte, *Point, error) {

	s, _, err := ec.RandomCurveScalar(rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	x, y := ec.Curve.ScalarBaseMult(s)
	return s, &Point{ec.Curve, x, y}, nil
}

func (ec *EC) RandomCurveScalar(rand io.Reader) ([]byte, *big.Int, error) {
	N := ec.Curve.Params().N // base point subgroup order
	bitLen := N.BitLen()
	byteLen := (bitLen + 7) >> 3
	buf := make([]byte, byteLen)

	// When in doubt, do what agl does in elliptic.go. Presumably
	// new(big.Int).SetBytes(b).Mod(N) would introduce bias, so we're sampling.
	for {
		_, err := io.ReadFull(rand, buf)
		if err != nil {
			return nil, nil, err
		}
		// Mask to account for field sizes that are not a whole number of bytes.
		buf[0] &= mask[bitLen%8]
		// Check if scalar is in the correct range.
		if new(big.Int).SetBytes(buf).Cmp(N) >= 0 {
			continue
		}
		break
	}

	return buf, new(big.Int).SetBytes(buf), nil
}

func (ec *EC) ScalarBaseMult(scalar *big.Int) *Point {
	x, y := ec.Curve.ScalarBaseMult(scalar.Bytes())
	return &Point{Curve: ec.Curve, X: x, Y: y}
}

func (ec *EC) ScalarMult(point *Point, scalar *big.Int) *Point {
	x, y := ec.Curve.ScalarMult(point.X, point.Y, scalar.Bytes())
	return &Point{Curve: ec.Curve, X: x, Y: y}
}

func (ec *EC) Add(pointA, pointB *Point) *Point {
	x, y := ec.Curve.Add(pointA.X, pointA.Y, pointB.X, pointB.Y)
	return &Point{Curve: ec.Curve, X: x, Y: y}
}

func (ec *EC) Inverse(pointA *Point) *Point {
	newPoint := &Point{
		X: new(big.Int).SetBytes(pointA.X.Bytes()),
		Y: new(big.Int).Sub(ec.Curve.Params().P, pointA.Y)}
	return newPoint
}

func (ec *EC) IsEqual(pointA, pointB *Point) bool {
	return pointA.X.Cmp(pointB.X) == 0 && pointA.Y.Cmp(pointB.Y) == 0
}

func (ec *EC) IsIdentity(pointA *Point) bool {
	pointB := ec.IdentityPoint()
	return ec.IsEqual(pointA, pointB)
}

func (p *Point) MarshalJSON() ([]byte, error) {
	byteRepr := p.Marshal()
	jsonRepr, err := json.Marshal(byteRepr)
	if err != nil {
		return nil, err
	}
	return jsonRepr, nil
}

func (p *Point) UnmarshalJSON(data []byte) error {
	var byteRepr []byte
	err := json.Unmarshal(data, &byteRepr)
	if err != nil {
		return err
	}
	return p.Unmarshal(p.Curve, byteRepr)
}

// Marshal calls through to elliptic.Marshal using the Curve field of the
// receiving Point. This produces an uncompressed marshaling as specified in
// SEC1 2.3.3.
func (p *Point) Marshal() []byte {
	return elliptic.Marshal(p.Curve, p.X, p.Y)
}

// Unmarshal interprets SEC1 2.3.4 compressed points in addition to the raw
// points supported by elliptic.Unmarshal. It assumes a NIST curve, and
// specifically that a = -3. It's faster when p = 3 mod 4 because of how
// ModSqrt works.
func (p *Point) Unmarshal(curve elliptic.Curve, data []byte) error {
	if curve == nil {
		return ErrUnspecifiedCurve
	}
	byteLen := (curve.Params().BitSize + 7) >> 3
	fieldOrder := curve.Params().P
	if len(data) == byteLen+1 {
		// Compressed point
		x := new(big.Int).SetBytes(data[1 : 1+byteLen])
		if x.Cmp(fieldOrder) != -1 {
			// x in [0, p-1]
			return ErrInvalidPoint
		}
		if data[0] == 0x02 || data[0] == 0x03 {
			sign := data[0] & 1 // "mod 2"

			// Recall y² = x³ - 3x + b
			// obviously, the Lsh trick is only valid when a = -3
			x3 := new(big.Int).Mul(x, x)          // x^2
			x3.Mul(x3, x)                         // x(x^2)
			threeTimesX := new(big.Int).Lsh(x, 1) // x << 1 == x*2
			threeTimesX.Add(threeTimesX, x)       // (x << 1) + x == x*3
			x3.Sub(x3, threeTimesX)               // x^3 - 3x
			x3.Add(x3, curve.Params().B)          // x^3 - 3x + b
			y := x3.ModSqrt(x3, fieldOrder)       // sqrt(x^3 - 3x + b) (mod p)
			if y == nil {
				// if no square root exists, either marshaling error
				// or an invalid curve point
				return ErrInvalidPoint
			}
			if sign != isOdd(y) {
				y.Sub(fieldOrder, y)
			}
			if !curve.IsOnCurve(x, y) {
				x = nil
				y = nil
				return ErrInvalidPoint
			}
			p.Curve = curve
			p.X, p.Y = x, y
			return nil
		}
		return ErrInvalidPoint
	}
	if len(data) == (2*byteLen)+1 && data[0] == 0x04 {
		// Uncompressed point
		p.Curve = curve
		p.X, p.Y = elliptic.Unmarshal(curve, data)
		if p.X == nil {
			return ErrInvalidPoint
		}
		return nil
	}
	return ErrInvalidPoint
}

func isOdd(x *big.Int) byte {
	return byte(x.Bit(0) & 1)
}

// BatchUnmarshalPoints takes a slice of P-256 curve points in the form specified
// in section 4.3.6 of ANSI X9.62 (see Go crypto/elliptic) and returns a slice
// of Point instances.
func BatchUnmarshalPoints(curve elliptic.Curve, data [][]byte) ([]*Point, error) {
	if curve == nil {
		return nil, ErrUnspecifiedCurve
	}
	decoded := make([]*Point, len(data))
	for i := 0; i < len(data); i++ {
		p := &Point{Curve: curve, X: nil, Y: nil}
		err := p.Unmarshal(curve, data[i])
		if err != nil {
			return nil, err
		}
		decoded[i] = p
	}
	return decoded, nil
}

// BatchMarshalPoints encodes a slice of Point objects in the form
// specified in section 4.3.6 of ANSI X9.62.
func BatchMarshalPoints(points []*Point) ([][]byte, error) {
	data := make([][]byte, len(points))
	for i := 0; i < len(points); i++ {
		data[i] = points[i].Marshal()
	}
	return data, nil
}

func (p *Point) String() string {
	return fmt.Sprintf("{X: %v, Y: %v}", p.X, p.Y)
}
