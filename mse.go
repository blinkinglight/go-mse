package mse

import (
	"fmt"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/nuid"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

var name string = "[noname]"
var channel string = "mse"
var cli *nats.Conn

var mu sync.Mutex
var nodes map[string]*node

type node struct {
	Start    int64
	Updated  int64
	Name     string
	IsMaster bool
}

func (n *node) Alive() bool {
	return time.Now().UnixNano()-n.Updated < time.Second.Nanoseconds()/10
}

var startTime int64
var connectTime int64

var IsConnected func() bool

func SetName(n string) {
	name = n
}

func SetChannel(c string) {
	channel = c
}

func Start() {
	startTime = time.Now().UnixNano()

	if IsConnected == nil {
		IsConnected = connected
	}

	if name == "[noname]" {
		name = nuid.New().Next()
	}

	nodes = make(map[string]*node)

	var e error
	cli, e = nats.Connect("nats://127.0.0.1:4222",
		nats.ReconnectHandler(func(_ *nats.Conn) {
			connectTime = time.Now().UnixNano()
		}),
		nats.Timeout(1*time.Second),
		nats.MaxReconnects(1<<31),
		nats.ReconnectWait(1*time.Second),
		nats.DontRandomize(),
	)

	if e != nil {
		log.Fatalf("nats connect error: %v", e)
	}
	connectTime = time.Now().UnixNano()

	go func() {
		for {
			time.Sleep(30 * time.Millisecond)
			if !cli.IsConnected() {
				continue
			}
			err := cli.Publish(channel, []byte(fmt.Sprintf("%v %v %v %v", "alive", name, startTime, IsMaster())))
			if err != nil {
				log.Printf("publish error: %v", err)
			}
			if e := cli.FlushTimeout(100 * time.Millisecond); e != nil {
				log.Printf("flush error: %v", e)
			}
		}
	}()

	cli.Subscribe(channel, func(msg *nats.Msg) {
		m := strings.Split(string(msg.Data), " ")
		switch len(m) {
		case 4:
			if m[0] == "alive" {
				mu.Lock()
				if _, ok := nodes[m[1]]; !ok {
					nodes[m[1]] = new(node)
				}
				num, _ := strconv.Atoi(m[2])
				nodes[m[1]].Start = int64(num)
				nodes[m[1]].Updated = time.Now().UnixNano()
				nodes[m[1]].Name = m[1]
				nodes[m[1]].IsMaster = (m[3] == "true")
				mu.Unlock()
			}
		case 2:
			if m[0] == "set-master" {
				min := startTime
				if m[1] == name {
					mu.Lock()
					for _, node := range nodes {
						if min > node.Start {
							min = node.Start
						}
					}
					startTime = min - 1
					mu.Unlock()
				}
			}
		}
	})
}

func IsMaster() bool {
	mu.Lock()

	min := startTime
	for _, node := range nodes {
		if !node.Alive() && IsConnected() {
			delete(nodes, node.Name)
		}
		if node.Start < min && node.Alive() {
			min = node.Start
		}

	}
	mu.Unlock()

	var uptime int64
	if IsConnected() {
		uptime = int64(time.Now().UnixNano() - startTime)
	} else {
		uptime = int64(0)
	}

	if uptime > int64(1*time.Second) {
		if IsConnected() {
			if startTime == min {
				return true
			}
		}
	}
	return false
}

func connected() bool {
	return cli.IsConnected()
}
