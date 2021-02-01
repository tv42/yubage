package pivplug

import (
	"crypto/sha256"
	"io"

	"eagain.net/go/yubage/internal/third_party/ageinternal"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

// encrypt[HKDF[salt, label]shared](file key) as per
// https://age-encryption.org/v1 just like X25519
//
// salt is ephemeral public key || public key,
// and label is "age-encryption.org/v1/piv-p256".

const wrapLabel = "age-encryption.org/v1/piv-p256"

func wrapKey(sharedSecret []byte, ephCompressed, pivCompressed []byte, key []byte) ([]byte, error) {
	salt := make([]byte, 0, len(ephCompressed)+len(pivCompressed))
	salt = append(salt, ephCompressed...)
	salt = append(salt, pivCompressed...)

	h := hkdf.New(sha256.New, sharedSecret, salt, []byte(wrapLabel))
	wrappingKey := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, wrappingKey); err != nil {
		return nil, err
	}

	wrappedKey, err := ageinternal.AEADEncrypt(wrappingKey, key)
	if err != nil {
		return nil, err
	}

	return wrappedKey, nil
}

func unwrapKey(sharedSecret []byte, ephCompressed, pivCompressed []byte, wrappedKey []byte) ([]byte, error) {
	salt := make([]byte, 0, len(ephCompressed)+len(pivCompressed))
	salt = append(salt, ephCompressed...)
	salt = append(salt, pivCompressed...)

	h := hkdf.New(sha256.New, sharedSecret, salt, []byte(wrapLabel))
	wrappingKey := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(h, wrappingKey); err != nil {
		return nil, err
	}

	// Assumption: file keys are forever 16 bytes.
	fileKey, err := ageinternal.AEADDecrypt(wrappingKey, 16, wrappedKey)
	if err != nil {
		return nil, err
	}

	return fileKey, nil
}
