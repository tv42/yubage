package pivplug_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"math/big"
	"testing"

	"eagain.net/go/yubage/internal/ageplugin"
	"eagain.net/go/yubage/internal/pivcard"
	"eagain.net/go/yubage/internal/pivcard/mock_pivcard"
	"eagain.net/go/yubage/internal/pivplug"
	"github.com/golang/mock/gomock"
	"github.com/google/go-cmp/cmp"
)

func mustParsePublicKey(t *testing.T, compressedBase64 string) *ecdsa.PublicKey {
	t.Helper()
	ephCompressed, err := base64.RawStdEncoding.Strict().DecodeString(compressedBase64)
	if err != nil {
		t.Fatalf("error parsing hardcoded public key: %v", err)
	}
	curve := elliptic.P256()
	x, y := elliptic.UnmarshalCompressed(curve, ephCompressed)
	if x == nil {
		t.Fatal("error uncompressing hardcoded P256 key")
	}
	pub := &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}
	return pub
}

func mustBigInt(t *testing.T, s string) *big.Int {
	i := new(big.Int)
	_, ok := i.SetString(s, 0)
	if !ok {
		t.Fatalf("cannot set big.Int: %q", s)
	}
	return i
}

func TestIdentityChatSimple(t *testing.T) {
	mocks := gomock.NewController(t)
	defer mocks.Finish()

	// $ go run ./internal/debug/cmd/generate-dummy-key/main.go
	// private                 54174045537741477645260415415255655016742280391432862109950881580092809591406
	// public,compr,b64        A2EY/MZxUdkdTAZbLn0Ly0GQGuyK58olRxAj8LghVSVe
	// recipient               age1yubikey1qds33lxxw9gaj82vqedjulgtedqeqxhv3tnu5f28zq3lpwpp25j4u9fu8kg
	// tag                     e2SWhQ
	private := &ecdsa.PrivateKey{
		PublicKey: *mustParsePublicKey(t, "A2EY/MZxUdkdTAZbLn0Ly0GQGuyK58olRxAj8LghVSVe"),
		D:         mustBigInt(t, "54174045537741477645260415415255655016742280391432862109950881580092809591406"),
	}

	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	// printf '\x01\x02\x03\x04\x82%s' "$(echo e2SWhQ==|base64 -d)"|bech32-encode AGE-PLUGIN-YUBIKEY-
	in.WriteString(`
-> add-identity AGE-PLUGIN-YUBIKEY-1QSPSYQVZ0DJFDPGWQ2RKZ

-> recipient-stanza 0 piv-p256 e2SWhQ AuXWo0GaigX07s5MpZ3O7W0LepaRgaQRZ8hcFzQyGPc5
fjpIzYC+PO66AJGLI2bU4k3Fg1CN+ysEcgGHg3WPpKE
-> done

-> ok

`[1:])

	// ephemeral public key from the above recipient-stanza
	ephPublic := mustParsePublicKey(t, "AuXWo0GaigX07s5MpZ3O7W0LepaRgaQRZ8hcFzQyGPc5")

	conn := ageplugin.New(in, out)
	cards := mock_pivcard.NewMockOpener(mocks)
	theCard := mock_pivcard.NewMockCard(mocks)
	gomock.InOrder()
	expectOpen := cards.EXPECT().
		Open(uint32(0x01020304), uint8(0x82)).
		Return(theCard, nil)
	theCard.EXPECT().
		Public().
		After(expectOpen).
		Return(private.Public())
	theCard.EXPECT().
		SharedKey(
			gomock.AssignableToTypeOf((*ecdsa.PublicKey)(nil)),
			gomock.AssignableToTypeOf(pivcard.Prompter(nil)),
		).
		After(expectOpen).
		DoAndReturn(func(peer *ecdsa.PublicKey, prompt pivcard.Prompter) ([]byte, error) {
			mult, _ := ephPublic.ScalarMult(ephPublic.X, ephPublic.Y, private.D.Bytes())
			secret := mult.Bytes()
			return secret, nil
		})
	theCard.EXPECT().
		Close().
		After(expectOpen)

	if err := pivplug.Identity(cards, conn); err != nil {
		t.Fatalf("pivplug.Identity: %v", err)
	}
	if in.Len() != 0 {
		t.Errorf("unconsumed input:\n%s", in.Bytes())
	}
	want := `
-> file-key 0
39MwXeehyuGJAvn2xYi48A
-> done

`[1:]
	got := out.String()
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("unexpected output (-got +want):\n%s", diff)
	}
}
