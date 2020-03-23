package ulog

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

// Test_LogFilter
func Test_LogFilter(t *testing.T) {
	var w []io.Writer
	nsqClient := NewNsqClient("ai-service", "eval", "192.168.6.100:4161")
	if nsqClient == nil {
		fmt.Printf("new nsql client fail\n")
		return
	}
	w = append(w, NewLogFilter(nsqClient), os.Stdout)
	l := New(append(w)...).SetLevel("info")
	l1 := l.With().Str("hostName", "bj01").Str("serverName", "eval01").Logger()
	l1.Info().Msgf("test")
	l1.Error().Msgf("test")
	l2 := l.With().Str("hostName", "bj01").Str("serverName", "eval01").Str("requestID", "requestIDTest").Str("globalID", "test22").Logger()
	//l2.Error().Msg("print1 error")
	l2.Info().Msg("print1 info")
	l2.Error().Msg("print1 error")
	time.Sleep(1 * time.Second)
	l2.Info().Msg("print2 info")
	time.Sleep(1 * time.Second)
	l2.Info().Msg("print3 info")
	l2.Error().Msg("print2 error")
	time.Sleep(5 * time.Second)
}
