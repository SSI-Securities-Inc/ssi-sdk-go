package signature

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math/big"
)

type rsaKeyValue struct {
	Modulus  string `xml:"Modulus"`
	Exponent string `xml:"Exponent"`
	P        string `xml:"P"`
	Q        string `xml:"Q"`
	DP       string `xml:"DP"`
	DQ       string `xml:"DQ"`
	InverseQ string `xml:"InverseQ"`
	D        string `xml:"D"`
}

var sha256DERPrefix = []byte{
	0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86,
	0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05,
	0x00, 0x04, 0x20,
}

func getRSAKey(privateKey string) (*big.Int, *big.Int, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	var key rsaKeyValue
	if err := xml.Unmarshal(keyBytes, &key); err != nil {
		return nil, nil, fmt.Errorf("failed to parse XML key: %w", err)
	}

	modBytes, err := base64.StdEncoding.DecodeString(key.Modulus)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode modulus: %w", err)
	}
	dBytes, err := base64.StdEncoding.DecodeString(key.D)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode D: %w", err)
	}

	n := new(big.Int).SetBytes(modBytes)
	d := new(big.Int).SetBytes(dBytes)
	return n, d, nil
}

// Sign creates a PKCS#1 v1.5 SHA-256 signature.
func Sign(data, privateKey string) (string, error) {
	n, d, err := getRSAKey(privateKey)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(data))
	digestInfo := append(sha256DERPrefix, hash[:]...)

	keyLen := (n.BitLen() + 7) / 8
	padLen := keyLen - len(digestInfo) - 3

	padded := make([]byte, keyLen)
	padded[0] = 0x00
	padded[1] = 0x01
	for i := 2; i < 2+padLen; i++ {
		padded[i] = 0xff
	}
	padded[2+padLen] = 0x00
	copy(padded[3+padLen:], digestInfo)

	m := new(big.Int).SetBytes(padded)
	s := new(big.Int).Exp(m, d, n)

	sigBytes := make([]byte, keyLen)
	sBytes := s.Bytes()
	copy(sigBytes[keyLen-len(sBytes):], sBytes)

	return fmt.Sprintf("%x", sigBytes), nil
}
