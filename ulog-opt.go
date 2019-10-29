package ulog

import (
	"io"
)

type uOptions struct {
	w          []io.Writer
	hostName   string
	serverName string
}

type Option interface {
	apply(*uOptions)
}

type optionFunc func(*uOptions)

func (f optionFunc) apply(o *uOptions) {
	f(o)
}

func AddWriter(w io.Writer) Option {
	return optionFunc(func(o *uOptions) {
		o.w = append(o.w, w)
	})
}

func HostName(hostName string) Option {
	return optionFunc(func(o *uOptions) {
		o.hostName = hostName
	})
}

func ServerName(serverName string) Option {
	return optionFunc(func(o *uOptions) {
		o.serverName = serverName
	})
}
