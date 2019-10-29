package ulog

import (
	"github.com/rs/zerolog"
	"io"
	//"github.com/rs/zerolog/log"
)

type Ulog struct {
	zerolog.Logger
}

func New(w ...io.Writer) *Ulog {
	return &Ulog{Logger: zerolog.New(io.MultiWriter(w...)).With().Timestamp().Logger()}
}

func (u *Ulog) Debug() *event {
	return &event{u.Logger.Debug()}
}

func (u *Ulog) Info() *event {
	return &event{u.Logger.Info()}
}

func (u *Ulog) Warn() *event {
	return &event{u.Logger.Warn()}
}

func (u *Ulog) Error() *event {
	return &event{u.Logger.Error()}
}

type event struct {
	*zerolog.Event
}

func (e *event) ID(id string) *event {
	return &event{e.Str("ID", id)}
}

func (e *event) IP(ip string) *event {
	return &event{e.Str("IP", ip)}
}
