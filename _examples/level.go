package main

import (
	"github.com/oraleval/ulog"
	"github.com/rs/zerolog"
	"os"
)

func main() {
	u := ulog.New([]ulog.Option{ulog.AddWriter(os.Stdout)}...)
	u.Logger = u.Level(zerolog.InfoLevel)

	u.Debug().Msg("hello")
	u.Error().Msg("hello")
	u.Warn().Msg("hello")
}
