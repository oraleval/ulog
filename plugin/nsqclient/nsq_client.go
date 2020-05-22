package nsqclient

import (
	"errors"
	"fmt"
	"github.com/guonaihong/gout"
	"github.com/nsqio/go-nsq"
	"io"
	"log"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

var _ io.Writer = (*nsqClient)(nil)

// nsq客户端
type nsqClient struct {
	message       chan []byte
	topic         string
	channel       string
	nsqLookupAddr string

	nsqNodeAddr []unsafe.Pointer
	curMaxIndex int32
	id          int64
}

// nsq客户端初始化函数
func NewNsqClient(topic, channel, nsqLookupAddr string) *nsqClient {
	n := &nsqClient{
		topic:         topic,
		channel:       channel,
		nsqLookupAddr: nsqLookupAddr,
		message:       make(chan []byte, 10000),
		nsqNodeAddr:   make([]unsafe.Pointer, 10), // 默认最多10个nsqd
	}

	go n.poll()
	return n
}

// write接口
func (q *nsqClient) Write(p []byte) (n int, err error) {
	select {
	case q.message <- append([]byte(nil), p...):
		return len(p), nil
	default:
	}

	return 0, errors.New("drop message")
}

// 获取nsqd 的地址列表
func (q *nsqClient) getNsqdAddr() ([]string, error) {
	//http: //192.168.6.100:4161/nodes
	p := producers{}
	err := gout.GET(q.nsqLookupAddr + "/nodes").BindJSON(&p).Do()
	if err != nil {
		return nil, err
	}

	if len(p.Producers) == 0 {
		return nil, errors.New("Node not found")
	}

	nodes := make([]string, len(p.Producers))
	for k, n := range p.Producers {
		a := strings.Split(n.Remote_address, ":")
		nodes[k] = fmt.Sprintf("%s:%d", a[0], n.Tcp_port)
	}

	return nodes, nil
}

// 更新nsqNode数组，如果nsqd地址有更新的,会直接动态扩容
// 目前updateNsqdAddr做了新增nsqd节点功能
// nsqd节点收缩功能放到poll函数里面，当写入失败，就会删除响应的nsqd节点
func (q *nsqClient) updateNsqdAddr(once bool) {

	for {

		time.Sleep(time.Second * 10)

		addrs, err := q.getNsqdAddr()
		if err != nil {
			log.Printf("get nsqd address fail:%s\n", err)
		}

		var newAddr []unsafe.Pointer
		// 从nsqlookup得到的nsqd列表
		for _, a := range addrs {
			found := false
			for j := int32(0); j < q.curMaxIndex; j++ {
				v := atomic.LoadPointer(&q.nsqNodeAddr[j])
				c := (*nsqNode)(v)
				if c == nil {
					continue
				}

				if c.addr == a {
					found = true
				}
			}

			if !found {
				newNsqNode, err := NewNsqNode(q, a)
				if err != nil {
					log.Printf("new nsq node fail:%s\n", err)
					continue
				}

				newAddr = append(newAddr, unsafe.Pointer(newNsqNode))
			}
		}

		// 处理新增
		if len(newAddr) > 0 {
			for _, v := range newAddr {
				if !q.setNodeAddr(v) {
					log.Printf("set nsq node fail:%v\n", (*nsqNode)(v))
				}
			}

			atomic.AddInt32(&q.curMaxIndex, int32(len(newAddr)))
		}

		//log.Printf("nsqd address:%v\n", addrs)
		//atomic.StorePointer(&q.nsqNodeAddr, unsafe.Pointer(&addrs))
		if once { //此判断用于测试
			break
		}
	}
}

// "192.168.6.100:4250"
func (q *nsqClient) nsqClient(addr string) (*nsq.Producer, error) {
	config := nsq.NewConfig()
	producer, err := nsq.NewProducer(addr, config)
	if err != nil {
		return nil, err
	}

	if err := producer.Ping(); err != nil {
		producer.Stop()
		return nil, err
	}

	return producer, nil
}

func (q *nsqClient) setNodeAddr(node unsafe.Pointer) (set bool) {
	for i := 0; i < len(q.nsqNodeAddr); i++ {
		if !atomic.CompareAndSwapPointer(&q.nsqNodeAddr[i], nil, node) {
			continue
		}
		//  成功设置到bucket里面

		for {
			// 最大小标小于当前值
			curMaxIndex := atomic.LoadInt32(&q.curMaxIndex)
			if curMaxIndex < int32(i) {
				if atomic.CompareAndSwapInt32(&q.curMaxIndex, curMaxIndex, int32(i)) {
					break
				}
				continue
			}
			break
		}

		return true
	}

	return false
}

func (q *nsqClient) initNsqNode() {
	addrs, err := q.getNsqdAddr()
	if err != nil {
		log.Printf("get nsqd addr fail:%s\n", err)
		return
	}

	index := int32(0)
	for _, v := range addrs {
		n, err := NewNsqNode(q, v)
		if err != nil {
			continue
		}

		if !q.setNodeAddr(unsafe.Pointer(n)) {
			log.Printf("set new node addr fail:address(%s)\n", v)
			continue
		}
		index++
	}
	atomic.StoreInt32(&q.curMaxIndex, index)
}

func (q *nsqClient) getProducer(index *int) *nsq.Producer {

	var v unsafe.Pointer
	var idx int64
	// 最多重试100次
	for i := 0; i < 100; i++ {
		curMaxIndex := atomic.LoadInt32(&q.curMaxIndex)
		if curMaxIndex > int32(len(q.nsqNodeAddr)) {
			curMaxIndex = int32(len(q.nsqNodeAddr))
		}

		idx = atomic.AddInt64(&q.id, 1) % int64(curMaxIndex)
		v = atomic.LoadPointer(&q.nsqNodeAddr[idx])
		if v != nil {
			break
		}
	}
	if v == nil {
		return nil
	}

	*index = int(idx)
	c := (*nsqNode)(v)
	return c.Producer
}

func (q *nsqClient) poll() {

	q.initNsqNode()
	go q.updateNsqdAddr(false)
	// 默认起4个go程
	for i := 0; i < 4; i++ {
		go func() {

			index := 0
			for m := range q.message {

				p := q.getProducer(&index)
				if p != nil {
					err := p.Publish(q.topic, m)
					if err == nil {
						// 成功写入
						continue
					}

					p.Stop()
					atomic.SwapPointer(&q.nsqNodeAddr[index], nil)
					log.Printf("write nsq fail:%s\n", p)
				}

				select {
				case q.message <- m:
				default:
					log.Printf("drop message :%s\n", m)
				}
			}

		}()
	}
}

// nsq node数据结构
type nsqNode struct {
	addr string // nsqd的地址
	*nsq.Producer
}

// 新建nsq node节点
func NewNsqNode(q *nsqClient, addr string) (*nsqNode, error) {
	p, err := q.nsqClient(addr)
	if err != nil {
		return nil, err
	}

	return &nsqNode{addr: addr, Producer: p}, nil
}

// 解析nsqd的地址列表
type producers struct {
	Producers []node `json:"producers"`
}

type node struct {
	Remote_address string `json:"remote_address"`
	Tcp_port       int    `json:"tcp_port"`
	Http_port      int    `json:"http_port"`
}
