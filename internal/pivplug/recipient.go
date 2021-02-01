package pivplug

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"

	"eagain.net/go/bech32"
	"eagain.net/go/yubage/internal/ageplugin"
)

func PublicKeyTagFromRecipient(recipient string) string {
	hashed := sha256.Sum256([]byte(recipient))
	tag := base64.RawStdEncoding.EncodeToString(hashed[:4])
	return tag
}

type PIVRecipient struct {
	Compressed []byte
	Public     *ecdsa.PublicKey
	Tag        string
}

const recipientHRP = "age1yubikey"

func ParsePIVRecipient(recipient string) (*PIVRecipient, error) {
	hrp, compressed, err := bech32.Decode(recipient)
	if err != nil {
		return nil, fmt.Errorf("cannot parse PIV recipient: %v", err)
	}
	if hrp != recipientHRP {
		return nil, errors.New("not a PIV recipient")
	}

	curve := elliptic.P256()
	x, y := elliptic.UnmarshalCompressed(curve, compressed)
	if x == nil {
		return nil, errors.New("does not contain a compressed P-256 key")
	}
	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
	tag := PublicKeyTagFromRecipient(recipient)

	r := &PIVRecipient{
		Compressed: compressed,
		Public:     pub,
		Tag:        tag,
	}
	return r, nil
}

func FormatPIVRecipient(compressed []byte) string {
	s, err := bech32.Encode(recipientHRP, compressed)
	if err != nil {
		// input data is fixed length, this just can't happen
		panic("Bech32 encode of ECDSA public key failed: " + err.Error())
	}
	return s
}

func Recipient(conn *ageplugin.Conn) error {
	debugf("recipient plugin start")
	defer debugf("recipient plugin stop")

	var (
		// these contain nil items for anything unrecognized, because
		// we have to use original indexes in responses

		recipients []string
		fileKeys   [][]byte
	)

loop:
	for {
		stanza, err := conn.ReadStanza()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read error: %v", err)
		}
		switch stanza.Type {
		case "add-recipient":
			// increase the count, no matter what
			recipients = append(recipients, "")
			if len(stanza.Args) != 1 {
				continue
			}
			if len(stanza.Body) != 0 {
				continue
			}
			recipients[len(recipients)-1] = stanza.Args[0]
		case "wrap-file-key":
			// increase the count, no matter what
			fileKeys = append(fileKeys, nil)
			if len(stanza.Args) != 0 {
				continue
			}
			fileKeys[len(fileKeys)-1] = stanza.Body
		case "done":
			if len(stanza.Args) != 0 {
				return errors.New("unexpected arguments in done stanza")
			}
			if len(stanza.Body) != 0 {
				return errors.New("unexpected body in done stanza")
			}
			break loop
		default:
			// ignore it
		}
	}

	for recipIdx, recip := range recipients {
		recipIdxStr := strconv.Itoa(recipIdx)

		pivRecipient, err := ParsePIVRecipient(recip)
		if err != nil {
			debugf("cannot parse as PIV recipient: %q: %v", recip, err)
			_ = err
			continue
		}

		for keyIdx, fileKey := range fileKeys {
			keyIdxStr := strconv.Itoa(keyIdx)

			eph, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			if err != nil {
				if err := conn.WriteStanza(&ageplugin.Stanza{
					Type: "error",
					Args: []string{"recipient", recipIdxStr},
					Body: []byte(fmt.Sprintf("generating ephemeral key failed: %v", err)),
				}); err != nil {
					return fmt.Errorf("writing wrap-file-key error response failed: %v", err)
				}
			}
			ephCompressed := elliptic.MarshalCompressed(eph.Curve, eph.PublicKey.X, eph.PublicKey.Y)
			ephCompressedStr := base64.RawStdEncoding.EncodeToString(ephCompressed)
			// ECDH shared secret between ephemeral key and yubikey
			sharedSecretNum, _ := eph.PublicKey.ScalarMult(pivRecipient.Public.X, pivRecipient.Public.Y, eph.D.Bytes())
			sharedSecret := sharedSecretNum.Bytes()

			wrappedKey, err := wrapKey(sharedSecret, ephCompressed, pivRecipient.Compressed, fileKey)
			if err != nil {
				return err
			}

			if err := conn.WriteStanza(&ageplugin.Stanza{
				Type: "recipient-stanza",
				Args: []string{keyIdxStr, "piv-p256", pivRecipient.Tag, ephCompressedStr},
				Body: wrappedKey,
			}); err != nil {
				return fmt.Errorf("writing wrap-file-key response failed: %v", err)
			}
		}
	}
	if err := conn.WriteStanza(&ageplugin.Stanza{
		Type: "done",
	}); err != nil {
		return fmt.Errorf("writing wrap-file-key response failed: %v", err)
	}
	return nil
}
