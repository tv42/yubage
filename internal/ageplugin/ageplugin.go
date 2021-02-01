// Package ageplugin talks the plugin side of the age plugin protocol.
package ageplugin

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
)

const (
	debug = false

	// setting this leaks ephemeral file keys and PINs into stderr/syslog
	debugStanzas = false
)

func debugf(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}

const (
	cmdPrefix = "-> "
)

type Conn struct {
	r *bufio.Reader
	w io.Writer
}

func New(r io.Reader, w io.Writer) *Conn {
	br := bufio.NewReader(r)
	c := &Conn{
		r: br,
		w: w,
	}
	return c
}

type Stanza struct {
	Type string
	Args []string
	Body []byte
}

func noEOF(err error) error {
	// age protocols include a well-defined shutdown and
	// are not terminated implicitly by EOF
	if err == io.EOF {
		return io.ErrUnexpectedEOF
	}
	return err
}

func (conn *Conn) ReadStanza() (*Stanza, error) {
	line, err := conn.r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read stanza: %w", noEOF(err))
	}
	if !strings.HasPrefix(line, cmdPrefix) {
		return nil, errors.New("no command recognized in input")
	}
	line = line[len(cmdPrefix):]
	line = strings.TrimSuffix(line, "\n")
	args := strings.Split(line, " ")
	cmd, args := args[0], args[1:]

	// Consume body. Construct an io.Reader to hand off to
	// base64.NewDecoder. We could be more clever, stream it while
	// feigning EOF at the right time -- but why bother, gather it
	// into an in-memory buffer line by line.
	//
	// Assumption: Bodies are small enough to read into memory.
	buf := new(bytes.Buffer)
	for {
		line, err := conn.r.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("reading line: %w", noEOF(err))
		}
		if len(line) > 64+1 {
			return nil, errors.New("line is too long")
		}
		_, _ = buf.Write(line)
		if len(line) < 64+1 {
			// Stanzas are terminated also by end of base64-encoded
			// data, detected as a partial line (or empty line). An
			// example complete stanza is `-> foo\nYmFy\n`, no final
			// empty line.
			//
			// This logic also detects empty lines.
			//
			// This is needed for when the parent switches to reading
			// from us, but is not consistently used to terminate all
			// bodies.
			break
		}
	}
	dec := base64.NewDecoder(base64.RawStdEncoding.Strict(), buf)
	body, err := ioutil.ReadAll(dec)
	if err != nil {
		return nil, err
	}
	s := &Stanza{
		Type: cmd,
		Args: args,
		Body: body,
	}
	if debugStanzas {
		debugf("read: %q %q %q", s.Type, s.Args, s.Body)
	}
	return s, nil
}

func (conn *Conn) WriteStanza(s *Stanza) error {
	// TODO validate outgoing Cmd & Args for character set, utf-8
	if debugStanzas {
		debugf("write: %q %q %q", s.Type, s.Args, s.Body)
	}
	buf := new(bytes.Buffer)
	buf.WriteString("-> ")
	buf.WriteString(s.Type)
	for _, arg := range s.Args {
		buf.WriteString(" ")
		buf.WriteString(arg)
	}
	buf.WriteString("\n")
	if _, err := conn.w.Write(buf.Bytes()); err != nil {
		return err
	}
	// TODO wrap lines at the correct column; won't trigger with
	// our small data sizes
	bodyWriter := base64.NewEncoder(base64.RawStdEncoding, conn.w)
	if _, err := bodyWriter.Write(s.Body); err != nil {
		return err
	}
	if err := bodyWriter.Close(); err != nil {
		return err
	}
	if _, err := io.WriteString(conn.w, "\n"); err != nil {
		return err
	}
	return nil
}

func (conn *Conn) Prompt(question string) (string, error) {
	if err := conn.WriteStanza(&Stanza{
		Type: "request-secret",
		Body: []byte(question),
	}); err != nil {
		return "", fmt.Errorf("writing request-secret failed: %v", err)
	}
	ok, err := conn.ReadStanza()
	if err != nil {
		return "", fmt.Errorf("reading request-secret response failed: %v", err)
	}
	if ok.Type != "ok" {
		return "", fmt.Errorf("bad request-secret response: %q", ok.Type)
	}
	if len(ok.Args) != 0 {
		return "", fmt.Errorf("bad request-secret response args: %#v", ok.Args)
	}
	response := string(ok.Body)
	return response, nil
}

func (conn *Conn) ReadOk() error {
	ok, err := conn.ReadStanza()
	if err != nil {
		return fmt.Errorf("cannot read result stanza: %v", err)
	}
	if ok.Type != "ok" {
		return fmt.Errorf("not ok: %q", ok.Type)
	}
	if len(ok.Args) != 0 {
		return fmt.Errorf("ok should not have arguments: %#v", ok.Args)
	}
	if len(ok.Body) != 0 {
		return fmt.Errorf("ok should not have body: %q", ok.Body)
	}
	return nil
}
