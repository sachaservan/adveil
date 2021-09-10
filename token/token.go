package token

import (
	"crypto/elliptic"

	"github.com/sachaservan/adveil/crypto"
)

type SignedBlindTokenWithMD struct {
	Curve elliptic.Curve
	W     *crypto.Point // signature
	MD    []byte        // public metadata
	Proof *crypto.Proof // for public metadata tokens
}

type SignedTokenWithMD struct {
	Curve elliptic.Curve
	T     []byte // token bytes
	Z     []byte // signature
	MD    []byte // public metadata
}

type SignedBlindToken struct {
	Curve elliptic.Curve
	B     *crypto.Point // blinded token
	W     *crypto.Point // signature
	S     []byte        // randomization value
}

type SignedToken struct {
	Curve elliptic.Curve
	T     []byte
	S     *crypto.Point
	W     *crypto.Point
}
