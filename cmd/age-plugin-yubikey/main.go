package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"log/syslog"
	"os"
	"os/signal"

	"eagain.net/go/yubage/internal/ageplugin"
	"eagain.net/go/yubage/internal/pivcard"
	"eagain.net/go/yubage/internal/pivplug"
	"golang.org/x/sys/unix"
)

// ignoreEPIPEWriter is a Writer that ignores EPIPE errors
// and discards the data.
type ignoreEPIPEWriter struct {
	w io.Writer
}

func (i *ignoreEPIPEWriter) Write(p []byte) (int, error) {
	n, err := i.w.Write(p)
	if errors.Is(err, unix.EPIPE) {
		return len(p), nil
	}
	return n, err
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("yubage: ")

	// TODO rage v0.5.0 eats plugin stderr, workaround by logging to
	// syslog. I wish I could just dup2 something over stderr, instead
	// of having to talk syslog protocol.
	logWriter, err := syslog.New(syslog.LOG_DEBUG|syslog.LOG_USER, "yubage")
	if err != nil {
		log.Fatalf("cannot open syslog: %v", err)
	}
	// logging to both so stderr is still there when running manually
	signal.Ignore(unix.SIGPIPE)
	log.SetOutput(io.MultiWriter(logWriter, &ignoreEPIPEWriter{os.Stderr}))
	defer func() {
		if err := recover(); err != nil {
			log.Printf("PANIC: %v", err)
			panic(err)
		}
	}()

	var agePlugin string
	flag.StringVar(&agePlugin, "age-plugin", "", "age plugin protocol to speak")

	flag.Parse()

	conn := ageplugin.New(os.Stdin, os.Stdout)
	switch agePlugin {
	case "identity-v1":
		cards := pivcard.New()
		if err := pivplug.Identity(cards, conn); err != nil {
			log.Fatal(err)
		}
	case "recipient-v1":
		if err := pivplug.Recipient(conn); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatal("unknown plugin")
	}
}
