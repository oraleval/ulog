package ulog

import (
	"fmt"
	"github.com/rs/zerolog"
	"io"
	"runtime"
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

func (u *Ulog) Error(skip ...int) *event {
	s := 1
	if len(skip) > 0 {
		s = skip[0] + 1
	}

	_, file, line, ok := runtime.Caller(s)
	if !ok {
		return &event{u.Logger.Error()}
	}

	return &event{u.Logger.Error().Str("stack", fmt.Sprintf("%s:%d", file, line))}
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
