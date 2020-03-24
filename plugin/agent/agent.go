package agent

import (
	"fmt"
	"github.com/guonaihong/gout"
	"io"
	"net/http"
	"sync/atomic"
)

const concurrency = 10

type Agent struct {
	agentAddr string
	*http.Client
	data    chan []byte
	running int32
}

type resp struct {
	ErrMsg string `json:"errmsg"`
}

var _ io.Writer = (io.Writer)(nil)

func New(agentAddr string) *Agent {
	a := &Agent{agentAddr: agentAddr}
	a.Client = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: concurrency,
		},
	}

	a.data = make(chan []byte, 1000)

	go a.poll()
	return a
}

func (a *Agent) Write(p []byte) (n int, err error) {
	select {
	case a.data <- append([]byte{}, p...):
	default:
		//drop message
	}
	return len(p), nil
}

func (a *Agent) IsRun() bool {
	return atomic.LoadInt32(&a.running) == 1
}

func (a *Agent) poll() {
	for i := 0; i < concurrency; i++ {
		go func() {
			atomic.CompareAndSwapInt32(&a.running, 0, 1)
			for d := range a.data {
				var r resp
				err := gout.POST(a.agentAddr + "/log-agent").Debug(true).SetBody(d).BindJSON(&r).Do()
				if err != nil {
					fmt.Printf("err:%s\n", err)
				}

				if r.ErrMsg != "" {
					fmt.Printf("errmsg:%s\n", r.ErrMsg)
				}
			}

		}()
	}
}
