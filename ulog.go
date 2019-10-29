package ulog

import (
	"github.com/rs/zerolog"
	"io"
	//"github.com/rs/zerolog/log"
)

type Ulog struct {
	zerolog.Logger
}

func New(o ...Option) *Ulog {
	options := uOptions{}
	for _, opt := range o {
		opt.apply(&options)
	}

	return &Ulog{Logger: zerolog.New(io.MultiWriter(options.w...)).With().Timestamp().
		Str("hostName", options.hostName).
		Str("serverName", options.serverName).
		Logger()}
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
