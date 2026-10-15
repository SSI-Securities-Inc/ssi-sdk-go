package signature

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math/big"
	"strings"
	"testing"

	ssisdk "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

func b64(i *big.Int) string { return base64.StdEncoding.EncodeToString(i.Bytes()) }

func xmlKey(t *testing.T, withExponent bool) (string, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	x := "<RSAKeyValue><Modulus>" + b64(priv.N) + "</Modulus>"
	if withExponent {
		x += "<Exponent>" + b64(big.NewInt(int64(priv.E))) + "</Exponent>"
	}
	x += "<D>" + b64(priv.D) + "</D></RSAKeyValue>"
	return base64.StdEncoding.EncodeToString([]byte(x)), priv
}

func TestSignMatchesStdlibPKCS1v15(t *testing.T) {
	for _, withE := range []bool{true, false} {
		key, priv := xmlKey(t, withE)
		got, err := Sign("hello world", key)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := hex.DecodeString(got)
		if err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256([]byte("hello world"))
		if err := rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, h[:], raw); err != nil {
			t.Fatalf("withExponent=%v: signature does not verify: %v", withE, err)
		}
		want, _ := rsa.SignPKCS1v15(nil, priv, crypto.SHA256, h[:])
		if string(raw) != string(want) {
			t.Fatalf("withExponent=%v: PKCS#1 v1.5 is deterministic, signatures must be equal", withE)
		}
	}
}

func TestSignBase64Encoding(t *testing.T) {
	key, priv := xmlKey(t, true)
	got, err := SignWithEncoding("payload", key, EncodingBase64)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256([]byte("payload"))
	if err := rsa.VerifyPKCS1v15(&priv.PublicKey, crypto.SHA256, h[:], raw); err != nil {
		t.Fatal(err)
	}
}

func TestBadKeyIsValidationErrorWithoutKeyMaterial(t *testing.T) {
	secretish := "c2VjcmV0LW5vdC1hLWtleQ==" // base64 of "secret-not-a-key"
	for _, k := range []string{"", "!!!not base64!!!", secretish} {
		_, err := Sign("x", k)
		var ve *ssisdk.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("key %q: want ValidationError, got %v", k, err)
		}
		if k != "" && strings.Contains(err.Error(), k) {
			t.Fatalf("error leaks key material: %v", err)
		}
	}
}

func TestUnknownEncoding(t *testing.T) {
	key, _ := xmlKey(t, true)
	if _, err := SignWithEncoding("x", key, "base32"); err == nil {
		t.Fatal("expected error")
	}
}
