package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"os"

	"eagain.net/go/yubage/internal/pivplug"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix(os.Args[0] + ": ")

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("private\t\t\t%v\n", key.D)
	compressed := elliptic.MarshalCompressed(key.Curve, key.PublicKey.X, key.PublicKey.Y)
	compressedStr := base64.RawStdEncoding.EncodeToString(compressed)
	fmt.Printf("public,compr,b64\t%s\n", compressedStr)
	recipient := pivplug.FormatPIVRecipient(compressed)
	fmt.Printf("recipient\t\t%s\n", recipient)
	fmt.Printf("tag\t\t\t%s\n", pivplug.PublicKeyTagFromRecipient(recipient))
}
