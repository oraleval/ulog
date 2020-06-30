package nsqclient

import (
	"bytes"
	"context"
	"fmt"
	"github.com/nsqio/go-nsq"
	"github.com/oraleval/ulog"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
	"unsafe"
)

// 先创建topic 和 channel
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
	err = consumer.ConnectToNSQLookupd(nsqLookUpAddr)
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
	c := NewNsqClient("my-test-topic", "my-test-channel", nsqLookUpAddr, 10000)
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

func NewNsqClientTest(topic, channel, nsqLookupAddr string) *nsqClient {
	n := &nsqClient{
		topic:         topic,
		channel:       channel,
		nsqLookupAddr: nsqLookupAddr,
		message:       make(chan []byte, 10000),
		nsqNodeAddr:   make([]unsafe.Pointer, 10), // 默认最多10个nsqd
	}
	return n
}

// 测试动态增加节点
// 测试过程:
// docker 新加一个nsqd节点
// 观察节点数有没有变多
func TestNsqClient_Add(t *testing.T) {
	c := NewNsqClientTest("my-test-topic", "my-test-channel", nsqLookUpAddr)
	c.initNsqNode()
	oldN := c.curMaxIndex
	c.updateNsqdAddr(true)
	newN := c.curMaxIndex
	assert.NotEqual(t, oldN, newN)
	fmt.Printf("old nsq number(%d), new nsq number(%d)\n", oldN, newN)
}

// 统计nsqd　节点个数
func countAddr(nsqNodeAddr []unsafe.Pointer) (count int) {
	for _, v := range nsqNodeAddr {
		if v != nil {
			count++
		}
	}
	return
}

// 测试删减节点
// 测试过程:
// docker 关闭一个nsqd节点
// 观察节点数有没有变多
func TestNsqClient_Del(t *testing.T) {
	c := NewNsqClientTest("my-test-topic", "my-test-channel", nsqLookUpAddr)
	go c.poll()
	time.Sleep(time.Second) //确保这时候nsqNodeAddr有值
	oldN := countAddr(c.nsqNodeAddr)
	fmt.Printf("stop nsqd\n")
	time.Sleep(time.Second * 10) //手动用docker新增一台nsqd节点
	for i := 0; i < 100; i++ {
		_, err := c.Write([]byte(fmt.Sprintf("hello%d", i)))
		assert.NoError(t, err)
	}

	time.Sleep(time.Second * 10) // 等待write删除挂掉的nsq节点
	newN := countAddr(c.nsqNodeAddr)
	fmt.Printf("old nsq number(%d), new nsq number(%d)\n", oldN, newN)
	assert.NotEqual(t, oldN, newN)
}

func TestNsqClient_Write2(t *testing.T) {
	nsqClient := NewNsqClient("ai-service", "eval", "192.168.6.100:4161", 10000)
	if nsqClient == nil {
		fmt.Printf("new nsql client fail\n")
		return
	}

	//time.Sleep(time.Second)

	//w := NewLogFilter(io.Writer(nsqClient))
	l := ulog.New(nsqClient, os.Stdout).SetLevel("info")
	l1 := l.With().Str("hostName", "bj01").Str("serverName", "eval01").Logger()
	l1.Info().Msgf("test")
	l1.Error().Msgf("test")

	l2 := l.With().Str("hostName", "bj01").Str("serverName", "eval01").Str("requestID", "requestIDTest").Str("globalID", "test1").Logger()
	//l2.Error().Msg("print1 error")
	l2.Info().Msg("print1 info")
	l2.Error().Msg("print1 error")
	l2.Info().Msg("print2 info")
	l2.Info().Msg("print3 info")
	l2.Error().Msg("print2 error")

	time.Sleep(time.Second)
}
