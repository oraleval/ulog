package ulog

import (
	"bytes"
	"context"
	"fmt"
	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// curl -X POST http://192.168.6.100:4151/topic/create?topic=my-test-topic
// curl -X POST 'http://192.168.6.100:4151/channel/create?topic=my-test-topic&channel=my-test-channel'

const nsqLookUpAddr = "192.168.6.100:4161"

type myMessageHandler struct {
	out chan []byte
}

// HandleMessage implements the Handler interface.
func (h *myMessageHandler) HandleMessage(m *nsq.Message) error {
	if len(m.Body) == 0 {
		// Returning nil will automatically send a FIN command to NSQ to mark the message as processed.
		fmt.Printf("body is empty\n")
		return nil
	}

	h.out <- m.Body
	fmt.Printf("#------------>:%s\n", m.Body)
	//err := processMessage(m.Body)

	// Returning a non-nil error will automatically send a REQ command to NSQ to re-queue the message.
	return nil
}

func nsqMockServer(ctx context.Context, out chan []byte) error {
	// Instantiate a consumer that will subscribe to the provided channel.
	config := nsq.NewConfig()
	config.MaxInFlight = 2
	consumer, err := nsq.NewConsumer("my-test-topic", "my-test-channel", config)
	if err != nil {
		return err
	}

	// Set the Handler for messages received by this Consumer. Can be called multiple times.
	// See also AddConcurrentHandlers.
	consumer.AddHandler(&myMessageHandler{
		out: out,
	})

	// Use nsqlookupd to discover nsqd instances.
	// See also ConnectToNSQD, ConnectToNSQDs, ConnectToNSQLookupds.
	err = consumer.ConnectToNSQLookupd("192.168.6.100:4161")
	if err != nil {
		return err
	}

	// Gracefully stop the consumer.

	for {
		select {
		case <-ctx.Done():
			return nil
		}
	}

	consumer.Stop()

	return err
}

// 测试读写
func TestNsqClient_Write(t *testing.T) {
	c := NewNsqClient("my-test-topic", "my-test-channel", nsqLookUpAddr)
	need := [][]byte{
		[]byte("test 1 message"),
		[]byte("test 2 message"),
	}

	for _, v := range need {
		_, err := c.Write(v)
		assert.NoError(t, err)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	out := make(chan []byte)
	go func() {
		nsqMockServer(ctx, out)
	}()

	readNumber := 0
	go func() {
	exit:
		for {
			select {
			case <-ctx.Done():
				break exit
			case m := <-out:
				have := false
				readNumber++
				for _, v := range need {
					if bytes.Equal(v, m) {
						have = true
						break
					}
				}
				assert.True(t, have, fmt.Sprintf("not found message"))
			}
		}
	}()

	<-ctx.Done()
	assert.Equal(t, len(need), readNumber)
}

// 测试动态增加节点

// 测试删减节点
