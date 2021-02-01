// Package pivcard provides a minimal abstraction over
// PIV card hardware token access, containing only the
// features needed by age-plugin-yubikey.
package pivcard

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"

	"github.com/go-piv/piv-go/piv"
)

const (
	debug = false
)

func debugf(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}

const (
	pivOrganization = "age-plugin-yubikey"
)

type Opener interface {
	Open(serial uint32, slot uint8) (Card, error)
}

type Prompter func(msg string) (string, error)

type Card interface {
	Close() error
	Public() *ecdsa.PublicKey
	SharedKey(peer *ecdsa.PublicKey, prompt Prompter) ([]byte, error)
}

type pivOpener struct{}

func New() Opener {
	return &pivOpener{}
}

var _ Opener = (*pivOpener)(nil)

func (o *pivOpener) Open(serial uint32, slot uint8) (Card, error) {
	// this takes uint32, but our recipient format only allows uint8,
	// and uint8 is the observed limit in the hardware
	pivSlot, ok := piv.RetiredKeyManagementSlot(uint32(slot))
	if !ok {
		return nil, fmt.Errorf("unrecognized slot: %02x", slot)
	}

	// the PCSC API is silly
	cards, err := piv.Cards()
	if err != nil {
		return nil, fmt.Errorf("cannot list PIV cards: %v", err)
	}
	for _, name := range cards {
		card, err := o.tryOpen(name, serial)
		if err != nil {
			debugf("ignoring card %q: %v", name, err)
			_ = err
			continue
		}

		// preload public key to simplify error handling
		cert, err := card.Certificate(pivSlot)
		if err != nil {
			debugf("ignoring card without certificate: %v", err)
			_ = err
			continue
		}

		orgs := cert.Subject.Organization
		if len(orgs) != 1 || orgs[0] != pivOrganization {
			debugf("ignoreing card with wrong organization: %q", orgs)
			continue
		}

		c := &pivCard{
			card:   card,
			serial: serial,
			slot:   pivSlot,
			pub:    cert.PublicKey.(*ecdsa.PublicKey),
		}
		return c, nil
	}
	return nil, errors.New("card not found")
}

func (o *pivOpener) tryOpen(name string, wantSerial uint32) (*piv.YubiKey, error) {
	card, err := piv.Open(name)
	if err != nil {
		return nil, fmt.Errorf("cannot open PIV card: %v", err)
	}
	defer func() {
		if card != nil {
			if err := card.Close(); err != nil {
				debugf("error closing PIV card: %v", err)
			}
		}
	}()

	gotSerial, err := card.Serial()
	if err != nil {
		return nil, fmt.Errorf("cannot get PIV card serial: %v", err)
	}
	if gotSerial != wantSerial {
		return nil, fmt.Errorf("unwanted serial: %08x", gotSerial)
	}

	tmp := card
	card = nil
	return tmp, nil
}

type pivCard struct {
	card   *piv.YubiKey
	serial uint32
	slot   piv.Slot
	pub    *ecdsa.PublicKey
}

var _ Card = (*pivCard)(nil)

func (c *pivCard) Close() error {
	return c.card.Close()
}

func (c *pivCard) Public() *ecdsa.PublicKey {
	return c.pub
}

func (c *pivCard) SharedKey(peer *ecdsa.PublicKey, prompt Prompter) ([]byte, error) {
	priv, err := c.card.PrivateKey(c.slot, c.pub, piv.KeyAuth{
		PINPrompt: func() (string, error) {
			return prompt(fmt.Sprintf("Enter PIN for Yubikey with serial %d", c.serial))
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get PIV private key handle: %v", err)
	}

	shared, err := priv.(*piv.ECDSAPrivateKey).SharedKey(peer)
	if err != nil {
		return nil, fmt.Errorf("PIV ECDHE error: %v", err)
	}
	return shared, nil
}
