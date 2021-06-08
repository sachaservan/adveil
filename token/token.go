package token

import "github.com/sachaservan/adveil/crypto"

type SignedToken struct {
	T     []byte        // token bytes
	W     *crypto.Point // signature
	MD    []byte        // public metadata
	Proof *crypto.Proof // for public metadata tokens
}
