# go-mse
golang master slave election
```
package main

import (
	"flag"
	"github.com/spx/go-mse"
	"log"
	"time"
)

var name = flag.String("n", "[noname]", "service name")

func main() {
	flag.Parse()

	mse.Start("master-slave", *name, "nats://127.0.0.1:4222")

	for {
		log.Printf("%v\n", mse.IsMaster())
		time.Sleep(200 * time.Millisecond)
	}
}
```

also listens on topic "[noname].rpc" and accepts "set-master" - switches node by name to master.
