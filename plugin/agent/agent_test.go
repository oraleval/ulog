package agent

import (
	"testing"
	"time"
)

func Test_Write(t *testing.T) {
	agent := New("127.0.0.1:8080")
	for !agent.IsRun() {
	}
	agent.Write([]byte("hello world 1\n"))
	agent.Write([]byte("hello world 2\n"))
	time.Sleep(time.Second)
}
