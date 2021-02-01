// Copyright 2019 Google LLC
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd

package ageinternal

import (
	"fmt"

	"golang.org/x/crypto/chacha20poly1305"
)

// AEADEncrypt encrypts a message with a one-time key.
func AEADEncrypt(key, plaintext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	// The nonce is fixed because this function is only used in places where the
	// spec guarantees each key is only used once (by deriving it from values
	// that include fresh randomness), allowing us to save the overhead.
	// For the code that encrypts the actual payload, look at the
	// filippo.io/age/internal/stream package.
	nonce := make([]byte, chacha20poly1305.NonceSize)
	return aead.Seal(nil, nonce, plaintext, nil), nil
}

// AEADDecrypt decrypts a message of an expected fixed size.
//
// The message size is limited to mitigate multi-key attacks, where a ciphertext
// can be crafted that decrypts successfully under multiple keys. Short
// ciphertexts can only target two keys, which has limited impact.
func AEADDecrypt(key []byte, size int, ciphertext []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) != size+aead.Overhead() {
		return nil, fmt.Errorf("encrypted message has unexpected length")
	}
	nonce := make([]byte, chacha20poly1305.NonceSize)
	return aead.Open(nil, nonce, ciphertext, nil)
}
