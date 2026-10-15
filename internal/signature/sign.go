package signature

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	ssisdk "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

// Encoding of the produced signature.
const (
	EncodingHex    = "hex"    // REST X-Signature
	EncodingBase64 = "base64" // trading WebSocket
)

type rsaKeyValue struct {
	Modulus  string `xml:"Modulus"`
	Exponent string `xml:"Exponent"`
	D        string `xml:"D"`
}

type rsaParsedKey struct {
	n, e, d *big.Int // e is nil when the XML has no Exponent
}

var (
	sha256DERPrefix = []byte{
		0x30, 0x31, 0x30, 0x0d, 0x06, 0x09, 0x60, 0x86,
		0x48, 0x01, 0x65, 0x03, 0x04, 0x02, 0x01, 0x05,
		0x00, 0x04, 0x20,
	}

	keyCache sync.Map // privateKey string -> *rsaParsedKey

	errInvalidKey = ssisdk.NewValidationError("private_key is not a valid base64-encoded XML RSA key")
)

func decodeInt(b64 string) (*big.Int, error) {
	b, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil || len(b) == 0 {
		return nil, errors.New("bad component")
	}
	return new(big.Int).SetBytes(b), nil
}

// parseKey parses the base64-encoded XML key. It never echoes key material or parser internals.
func parseKey(privateKey string) (*rsaParsedKey, error) {
	if cached, ok := keyCache.Load(privateKey); ok {
		return cached.(*rsaParsedKey), nil
	}
	keyBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return nil, errInvalidKey
	}
	var kv rsaKeyValue
	if err := xml.Unmarshal(keyBytes, &kv); err != nil {
		return nil, errInvalidKey
	}
	n, err := decodeInt(kv.Modulus)
	if err != nil {
		return nil, errInvalidKey
	}
	d, err := decodeInt(kv.D)
	if err != nil {
		return nil, errInvalidKey
	}
	k := &rsaParsedKey{n: n, d: d}
	if strings.TrimSpace(kv.Exponent) != "" {
		if e, err := decodeInt(kv.Exponent); err == nil {
			k.e = e
		}
	}
	if n.BitLen() < 512 {
		return nil, errInvalidKey
	}
	keyCache.Store(privateKey, k)
	return k, nil
}

// Sign creates a PKCS#1 v1.5 SHA-256 signature, hex encoded (REST X-Signature).
func Sign(data, privateKey string) (string, error) {
	return SignWithEncoding(data, privateKey, EncodingHex)
}

// SignWithEncoding signs data and encodes the signature as "hex" or "base64".
// A missing or malformed key is a ValidationError that never echoes key material.
func SignWithEncoding(data, privateKey, encoding string) (string, error) {
	if encoding != EncodingHex && encoding != EncodingBase64 {
		return "", ssisdk.NewValidationError(fmt.Sprintf("signature encoding must be 'hex' or 'base64', got %q", encoding))
	}
	if privateKey == "" {
		return "", ssisdk.NewValidationError("private_key is required to sign this request")
	}
	k, err := parseKey(privateKey)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256([]byte(data))
	digestInfo := make([]byte, 0, len(sha256DERPrefix)+len(hash))
	digestInfo = append(digestInfo, sha256DERPrefix...)
	digestInfo = append(digestInfo, hash[:]...)

	keyLen := (k.n.BitLen() + 7) / 8
	padLen := keyLen - len(digestInfo) - 3
	if padLen < 8 {
		return "", errInvalidKey
	}
	padded := make([]byte, keyLen)
	padded[0] = 0x00
	padded[1] = 0x01
	for i := 2; i < 2+padLen; i++ {
		padded[i] = 0xff
	}
	padded[2+padLen] = 0x00
	copy(padded[3+padLen:], digestInfo)

	m := new(big.Int).SetBytes(padded)
	s, err := privateOp(m, k)
	if err != nil {
		return "", err
	}
	sigBytes := make([]byte, keyLen)
	sBytes := s.Bytes()
	copy(sigBytes[keyLen-len(sBytes):], sBytes)

	if encoding == EncodingBase64 {
		return base64.StdEncoding.EncodeToString(sigBytes), nil
	}
	return hex.EncodeToString(sigBytes), nil
}

// privateOp is the RSA private-key operation with base blinding and a result check. Blinding
// randomizes the value d is applied to, which blunts timing attacks. With the public exponent
// known the signature is verified before it is returned, so a faulty computation can never
// leak a bad signature.
func privateOp(m *big.Int, k *rsaParsedKey) (*big.Int, error) {
	if k.e == nil {
		return new(big.Int).Exp(m, k.d, k.n), nil
	}
	one := big.NewInt(1)
	bound := new(big.Int).Sub(k.n, big.NewInt(3))
	var r, rInv *big.Int
	for {
		x, err := rand.Int(rand.Reader, bound)
		if err != nil {
			return nil, ssisdk.NewValidationError("signing failed: no randomness available")
		}
		r = x.Add(x, big.NewInt(2))
		if new(big.Int).GCD(nil, nil, r, k.n).Cmp(one) == 0 {
			rInv = new(big.Int).ModInverse(r, k.n)
			if rInv != nil {
				break
			}
		}
	}
	blinded := new(big.Int).Exp(r, k.e, k.n)
	blinded.Mul(blinded, m).Mod(blinded, k.n)
	s := new(big.Int).Exp(blinded, k.d, k.n)
	s.Mul(s, rInv).Mod(s, k.n)
	if new(big.Int).Exp(s, k.e, k.n).Cmp(m) != 0 {
		return nil, ssisdk.NewValidationError("signing failed an internal consistency check")
	}
	return s, nil
}
