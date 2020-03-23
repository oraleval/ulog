package main

import (
	"github.com/oraleval/ulog"
	"github.com/rs/zerolog"
	"os"
	"time"
)

func main() {

	zerolog.TimeFieldFormat = time.RFC3339Nano
	u := ulog.New(os.Stdout)
	u.Logger = u.Level(zerolog.InfoLevel)

	u.Debug().Msg("hello")
	u.Error().Msg("hello")
	u.Warn().Msg("hello")
}
