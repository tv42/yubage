package ageplugin_test

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"eagain.net/go/yubage/internal/ageplugin"
	"github.com/google/go-cmp/cmp"
)

func TestReadStanza(t *testing.T) {
	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	conn := ageplugin.New(in, out)
	in.WriteString("-> foo bar baz\ndGh1ZA\n")
	got, err := conn.ReadStanza()
	if err != nil {
		t.Errorf("ReadStanza: %v", err)
	}
	want := &ageplugin.Stanza{
		Type: "foo",
		Args: []string{"bar", "baz"},
		Body: []byte("thud"),
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong stanza (-got +want)\n%s", diff)
	}
	if in.Len() != 0 {
		t.Errorf("unconsumed input:\n%s", in.Bytes())
	}
	if out.Len() != 0 {
		t.Errorf("unexpected output:\n%s", out.Bytes())
	}
}

func TestReadStanzaShort(t *testing.T) {
	const input = "-> foo bar baz\ndGh1ZA"
	for n := 0; n <= len(input); n++ {
		s := input[:n]
		t.Run(s, func(t *testing.T) {
			in := new(bytes.Buffer)
			out := new(bytes.Buffer)
			conn := ageplugin.New(in, out)
			in.WriteString(s)
			got, err := conn.ReadStanza()
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("bad error: %v", err)
			}
			if got != nil {
				t.Errorf("bad stanza: %+v", got)
			}
			if out.Len() != 0 {
				t.Errorf("unexpected output:\n%s", out.Bytes())
			}
		})
	}
}

func TestWriteStanza(t *testing.T) {
	in := new(bytes.Buffer)
	out := new(bytes.Buffer)
	conn := ageplugin.New(in, out)
	err := conn.WriteStanza(&ageplugin.Stanza{
		Type: "foo",
		Args: []string{"bar", "baz"},
		Body: []byte("thud"),
	})
	if err != nil {
		t.Fatalf("WriteStanza: %v", err)
	}
	got := out.String()
	want := "-> foo bar baz\ndGh1ZA\n"
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("wrong stanza (-got +want)\n%s", diff)
	}
}
