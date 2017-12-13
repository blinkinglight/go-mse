package mse

import (
	"fmt"
	"github.com/nats-io/go-nats"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	Start   int64
	Updated int64
}

func (n *Node) IsAlive() bool {
	return time.Now().UnixNano()-n.Updated < time.Second.Nanoseconds()/10
}

type As struct {
	start      int64
	name       string
	master     bool
	alive      bool
	gotMsg     bool
	msgTime    int64
	nodes      map[string]*Node
	mu         sync.Mutex
	masterName string
}

func (a *As) Set(name string, start int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.nodes[name]; !ok {
		a.nodes[name] = new(Node)
	}
	a.nodes[name].Start = start
	a.nodes[name].Updated = time.Now().UnixNano()

}

func (a *As) Update() {
	if !a.IsAlive() {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	start := as.start
	masterName := as.name
	for name, node := range a.nodes {
		if node.IsAlive() {
			if node.Start < start {
				start = node.Start
				masterName = name
			}
		}
	}
	as.master = as.start <= start
	as.masterName = masterName
}

func (a *As) SetSelfMaster() {
	a.mu.Lock()
	defer a.mu.Unlock()
	start := a.start
	for _, node := range a.nodes {
		if node.Start < start {
			start = node.Start
		}
	}
	if start < a.start {
		as.start = start - 1
	}
}

func (a *As) IsMaster() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.master
}

func (a *As) IsSlave() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return !a.master
}
func (a *As) IsAlive() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.alive
}

var as *As

func IsMaster() bool {
	return as.IsMaster()
}

func IsSlave() bool {
	return as.IsSlave()
}

func IsAlive() bool {
	return as.IsAlive()
}

func MasterName() string {
	return as.masterName
}

func SetRPCChannel(name string) {
	rpcChannel = name
}

func DoRCP(name string, data string, timeout int) (string, error) {
	msg, err := cli.Request(name, []byte(data), time.Duration(timeout)*time.Millisecond)
	if err != nil {
		return "", err
	}
	return string(msg.Data), nil
}

var OnPC func(string, string)

var rpcChannel string

var cli *nats.Conn

func Start(topic, name, natsServer string) {

	as = new(As)
	as.nodes = make(map[string]*Node)
	start := time.Now().UnixNano()
	as.start = start
	as.name = name
	var e error
	cli, e = nats.Connect(natsServer)
	if e != nil {
		panic(e)
	}
	as.alive = cli.IsConnected()

	if rpcChannel == "" {
		rpcChannel = name + ".rpc"
	}

	cli.Subscribe(rpcChannel, func(msg *nats.Msg) {
		data := string(msg.Data)
		if data == "set-master" {
			as.SetSelfMaster()
		} else {
			params := strings.SplitN(data, " ", 2)
			if len(params) < 2 {
				OnPC(params[0], "")
			} else {
				OnPC(params[0], params[1])
			}
			cli.Publish(msg.Reply, []byte("OK"))
		}
	})
	go func() {
		for {
			as.mu.Lock()
			cli.Publish(topic, []byte(fmt.Sprintf("%v %v %d", "alive", name, as.start)))
			as.mu.Unlock()
			as.alive = cli.IsConnected()
			time.Sleep(30 * time.Millisecond)
		}
	}()
	go func() {
		time.Sleep(500 * time.Millisecond)
		if as.gotMsg == false && as.IsAlive() {
			as.master = true
		}
	}()
	go func() {
		for {
			time.Sleep(30 * time.Millisecond)
			as.Update()
		}
	}()

	go func() {
		cli.Subscribe(topic, func(msg *nats.Msg) {
			m := strings.Split(string(msg.Data), " ")
			if m[0] == "alive" && m[1] != name {
				as.gotMsg = true
				as.msgTime = time.Now().UnixNano()
				num, _ := strconv.Atoi(m[2])
				as.Set(m[1], int64(num))
			}
		})
	}()

}
