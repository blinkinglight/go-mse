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
	start   int64
	name    string
	master  bool
	alive   bool
	gotMsg  bool
	msgTime int64
	nodes   map[string]*Node
	mu      sync.Mutex
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
	for _, node := range a.nodes {
		if node.IsAlive() {
			if node.Start < start {
				start = node.Start
			}
		}
	}
	as.master = as.start <= start

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

func Start(topic, name, natsServer string) {

	as = new(As)
	as.nodes = make(map[string]*Node)
	start := time.Now().UnixNano()
	as.start = start
	as.name = name

	cli, e := nats.Connect(natsServer)
	if e != nil {
		panic(e)
	}
	as.alive = cli.IsConnected()

	cli.Subscribe(name+".rpc", func(msg *nats.Msg) {
		if string(msg.Data) == "set-master" {
			as.SetSelfMaster()
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
