package logger

import (
	"log"
	"os"
)

func New(prefix string) *log.Logger {
	if prefix != "" {
		prefix += " "
	}
	return log.New(os.Stdout, prefix, log.LstdFlags)
}
