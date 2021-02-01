package pivplug_test

import (
	"bytes"
	"regexp"
	"testing"

	"eagain.net/go/yubage/internal/ageplugin"
	"eagain.net/go/yubage/internal/pivplug"
)

func TestRecipientChatSimple(t *testing.T) {
	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	// $ go run ./internal/debug/cmd/generate-dummy-key/main.go
	// private                 54174045537741477645260415415255655016742280391432862109950881580092809591406
	// public,compr,b64        A2EY/MZxUdkdTAZbLn0Ly0GQGuyK58olRxAj8LghVSVe
	// recipient               age1yubikey1qds33lxxw9gaj82vqedjulgtedqeqxhv3tnu5f28zq3lpwpp25j4u9fu8kg
	// tag                     e2SWhQ
	in.WriteString(`
-> add-recipient age1yubikey1qds33lxxw9gaj82vqedjulgtedqeqxhv3tnu5f28zq3lpwpp25j4u9fu8kg

-> wrap-file-key
39MwXeehyuGJAvn2xYi48A
-> done

`[1:])
	conn := ageplugin.New(in, out)
	if err := pivplug.Recipient(conn); err != nil {
		t.Fatalf("pivplug.Recipient: %v", err)
	}
	if in.Len() != 0 {
		t.Errorf("unconsumed input:\n%s", in.Bytes())
	}
	want := regexp.MustCompile(`
^-> recipient-stanza 0 piv-p256 e2SWhQ [A-Za-z0-9+/]{44}
[A-Za-z0-9+/]{43}
-> done

$`[1:])
	got := out.Bytes()
	t.Logf("got\n%s", got)
	if !want.Match(got) {
		t.Errorf("unexpected output:\n%s", got)
	}
}
