package pivplug

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"eagain.net/go/bech32"
	"eagain.net/go/yubage/internal/ageplugin"
	"eagain.net/go/yubage/internal/pivcard"
)

type PIVIdentity struct {
	Serial uint32
	Slot   uint8
	Tag    string
}

func ParsePIVIdentity(ident string) (*PIVIdentity, error) {
	hrp, data, err := bech32.Decode(ident)
	if err != nil {
		return nil, err
	}
	if hrp != "AGE-PLUGIN-YUBIKEY-" {
		return nil, errors.New("wrong recipient type")
	}
	if got := len(data); got != 4+1+4 {
		return nil, fmt.Errorf("wrong data length: %d", got)
	}
	tagBuf := data[5:9]
	tag := base64.RawStdEncoding.EncodeToString(tagBuf)
	id := &PIVIdentity{
		Serial: binary.LittleEndian.Uint32(data[:4]),
		Slot:   data[4],
		Tag:    tag,
	}
	return id, nil
}

type pivRecipientStanza struct {
	Index          string
	Tag            string
	EphCompressed  []byte
	WrappedFileKey []byte
}

func Identity(pivcards pivcard.Opener, conn *ageplugin.Conn) error {
	debugf("identity plugin start")
	defer debugf("identity plugin stop")

	var (
		// these contain nil items for anything unrecognized, because
		// we have to use original indexes in responses

		identities []*PIVIdentity
		recipients []*pivRecipientStanza
	)

loop:
	for {
		stanza, err := conn.ReadStanza()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("receive error: %v", err)
		}
		switch stanza.Type {
		case "add-identity":
			// increase the count, no matter what
			identities = append(identities, nil)
			if len(stanza.Args) != 1 {
				continue
			}
			if len(stanza.Body) != 0 {
				continue
			}
			id, err := ParsePIVIdentity(stanza.Args[0])
			if err != nil {
				debugf("error parsing PIV identity: %v", err)
				_ = err
				continue
			}
			identities[len(identities)-1] = id
		case "recipient-stanza":
			// increase the count, no matter what
			recipients = append(recipients, nil)
			if len(stanza.Args) != 4 {
				continue
			}
			if stanza.Args[1] != "piv-p256" {
				continue
			}
			tag := stanza.Args[2]
			ephCompressed, err := base64.RawStdEncoding.Strict().DecodeString(stanza.Args[3])
			if err != nil {
				debugf("error parsing public key in recipient-stanza: %v", err)
				_ = err
				continue
			}
			recipients[len(recipients)-1] = &pivRecipientStanza{
				Index:          stanza.Args[0],
				Tag:            tag,
				EphCompressed:  ephCompressed,
				WrappedFileKey: stanza.Body,
			}
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

	for _, recip := range recipients {
		if recip == nil {
			continue
		}
		curve := elliptic.P256()
		x, y := elliptic.UnmarshalCompressed(curve, recip.EphCompressed)
		if x == nil {
			return errors.New("cannot unmarshal P256 key")
		}
		ephPub := &ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		}

		for _, ident := range identities {
			if ident == nil {
				continue
			}
			if recip.Tag != ident.Tag {
				debugf("ident mismatch %#v vs %#v", recip, ident)
				continue
			}

			card, err := pivcards.Open(ident.Serial, ident.Slot)
			if err != nil {
				debugf("cannot open PIV card: %v", err)
				_ = err
				continue
			}
			defer func() {
				if err := card.Close(); err != nil {
					debugf("error closing card: %v", err)
					_ = err
				}
			}()

			pivPublicKey := card.Public()
			pivCompressed := elliptic.MarshalCompressed(pivPublicKey.Curve, pivPublicKey.X, pivPublicKey.Y)

			// Compare tag again, to avoid unnecessarily prompting
			// for PINs in case the identity is stale data
			//
			// The PIV-P256 format tag is defined in terms of the recipient string,
			// not the public key. Need to encode the key from hardware to get the
			// correct	 tag.
			tag := PublicKeyTagFromRecipient(FormatPIVRecipient(pivCompressed))
			if tag != ident.Tag {
				debugf("stale tag: %q != %q", tag, ident.Tag)
				continue
			}

			sharedSecret, err := card.SharedKey(ephPub, conn.Prompt)
			if err != nil {
				debugf("shared secret error: %v", err)
				_ = err
				continue
			}

			fileKey, err := unwrapKey(sharedSecret, recip.EphCompressed, pivCompressed, recip.WrappedFileKey)
			if err != nil {
				debugf("aead decrypt: %v", err)
				_ = err
				continue
			}

			if err := conn.WriteStanza(&ageplugin.Stanza{
				Type: "file-key",
				Args: []string{recip.Index},
				Body: []byte(fileKey),
			}); err != nil {
				return fmt.Errorf("writing file-key response failed: %v", err)
			}
			if err := conn.ReadOk(); err != nil {
				return fmt.Errorf("file-key error: %v", err)
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
