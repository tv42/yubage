package pivplug

import "log"

const (
	debug = true
)

func debugf(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}
