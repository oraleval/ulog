package filter

import (
	"fmt"
	"github.com/oraleval/ulog"
	"github.com/oraleval/ulog/plugin/nsqclient"
	"io"
	"os"
	"testing"
	"time"
)

// Test_LogFilter
func Test_LogFilter(t *testing.T) {
	var w []io.Writer
	client := nsqclient.NewNsqClient("ai-service", "eval", "192.168.6.100:4161")
	if client == nil {
		fmt.Printf("new nsql client fail\n")
		return
	}
	t1 := NewLogFilter(client, "/tmp/fileDB")
	w = append(w, t1)
	l := ulog.New(append(w)...).SetLevel("info")
	l2 := l.With().Str("hostName", "bj01").Str("serverName", "eval01").Str("requestID", "requestIDTest").Str("globalID", "test22").Logger()
	//l2.Error().Msg("print1 error")
	l2.Info().Msg("print1 info")
	l2.Error().Msg("print1 error")
	time.Sleep(1 * time.Second)
	l2.Info().Msg("print2 info")
	time.Sleep(1 * time.Second)
	l2.Info().Msg("print3 info")
	l2.Error().Msg("print2 error")

	//list, _ := t1.GetLogCache("test22")
	//fmt.Println(list)
	time.Sleep(1000 * time.Second)
}

func Test_File(t *testing.T) {
	f := NewLogFilter(os.Stdout, "/tmp/fileDB")
	list, _ := f.GetLogCache("test22")
	for _, i := range list {
		fmt.Println(string(i))
	}
}
