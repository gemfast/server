package utils

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
)

func VerifySignature(algorithm string, publicKey, signed, candidate []byte) error {
	parser, err := parsePublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("could not load publick key: %v", err)
	}
	err = parser.Unsign(algorithm, signed, candidate)
	if err != nil {
		return fmt.Errorf("could not verify request signature: %v", err)
	}
	return nil
}

// parsePublicKey parses a PEM encoded private key.
func parsePublicKey(pemBytes []byte) (Unsigner, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("ssh: no key found")
	}

	var rawkey interface{}
	switch block.Type {
	case "PUBLIC KEY":
		rsa, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		rawkey = rsa
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %q", block.Type)
	}

	return newUnsignerFromKey(rawkey)
}

// A Signer is can create signatures that verify against a public key.
type Unsigner interface {
	// Sign returns raw signature for the given data. This method
	// will apply the hash specified for the keytype to the data.
	Unsign(algorithm string, data []byte, sig []byte) error
}

func newUnsignerFromKey(k interface{}) (Unsigner, error) {
	var sshKey Unsigner
	switch t := k.(type) {
	case *rsa.PublicKey:
		sshKey = &rsaPublicKey{t}
	default:
		return nil, fmt.Errorf("ssh: unsupported key type %T", k)
	}
	return sshKey, nil
}

type rsaPublicKey struct {
	*rsa.PublicKey
}

// Unsign verifies the message using a rsa-sha256 signature
func (r *rsaPublicKey) Unsign(algorithm string, message []byte, sig []byte) error {
	if algorithm == "sha1" {
		h := sha1.New()
		h.Write(message)
		d := h.Sum(nil)
		return rsa.VerifyPKCS1v15(r.PublicKey, crypto.SHA1, d, sig)
	}
	h := sha256.New()
	h.Write(message)
	d := h.Sum(nil)
	return rsa.VerifyPKCS1v15(r.PublicKey, crypto.SHA256, d, sig)
}
