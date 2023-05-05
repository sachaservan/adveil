package ec

import (
	"crypto/elliptic"
	"testing"
)

func TestIdentity(t *testing.T) {

	ec := &EC{elliptic.P256()}
	I := ec.IdentityPoint()
	_, R, _ := ec.NewRandomPoint()
	sum := ec.Add(I, R)

	if !ec.IsEqual(R, sum) {
		t.Fatalf("Identity point is not correct")
	}
}

func TestAdd(t *testing.T) {

	ec := &EC{elliptic.P256()}
	_, R1, _ := ec.NewRandomPoint()
	_, R2, _ := ec.NewRandomPoint()

	x, y := ec.Curve.Add(R1.X, R1.Y, R2.X, R2.Y)
	P := &Point{
		X: x,
		Y: y,
	}

	sum := ec.Add(R1, R2)

	if !ec.IsEqual(P, sum) {
		t.Fatalf("Add is wrong")
	}
}

func TestInverse(t *testing.T) {

	ec := &EC{elliptic.P256()}
	_, R, _ := ec.NewRandomPoint()
	I := ec.IdentityPoint()

	inv := ec.Inverse(R)
	res := ec.Add(R, inv)

	if !ec.IsEqual(res, I) {
		t.Fatalf("Inverse is wrong")
	}
}

func BenchmarkCurveAddition(b *testing.B) {

	ec := &EC{elliptic.P256()}

	list := make([]*Point, 1000)
	for i := 0; i < 1000; i++ {
		_, r, _ := ec.NewRandomPoint()
		list[i] = r
	}

	b.ResetTimer()

	next := 0
	for i := 0; i < b.N; i++ {
		if next+2 > len(list) {
			next = 0
		}

		ec.Add(list[next], list[next+1])
		next++
	}
}

func TestMarshall(t *testing.T) {

	ec := &EC{elliptic.P256()}
	_, P, _ := ec.NewRandomPoint()
	data := P.Marshal()
	R := ec.IdentityPoint()
	err := R.Unmarshal(ec.Curve, data)
	if err != nil {
		t.Fatal(err)
	}

	if !ec.IsEqual(R, P) {
		t.Fatalf("Recovered point is not correct")
	}
}

func TestMarshallJSON(t *testing.T) {

	ec := &EC{elliptic.P256()}
	_, P, _ := ec.NewRandomPoint()

	data, err := P.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}

	R := ec.IdentityPoint()
	err = R.UnmarshalJSON(data)
	if err != nil {
		t.Fatal(err)
	}

	if !ec.IsEqual(R, P) {
		t.Fatalf("Recovered point is not correct")
	}
}
