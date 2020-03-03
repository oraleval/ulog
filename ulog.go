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

// 初始化函数
func New(w ...io.Writer) *Ulog {
	return &Ulog{Logger: zerolog.New(io.MultiWriter(w...)).With().Timestamp().
		Logger()}
}

// 设置日志等级
func (u *Ulog) SetLevel(level string) *Ulog {
	l, err := zerolog.ParseLevel(level)
	if err != nil {
		panic(err)
	}

	u.Logger = u.Level(l)
	return u
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
