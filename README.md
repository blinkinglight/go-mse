# go-mse
golang master slave election
```
package main

import (
	"github.com/spx/go-mse"
	"log"
	"time"
)

func main() {
	mse.SetChannel("mse")
	mse.SetNatsServer("nats://127.0.0.1:4222")
	// mse.SetName("myfirstnode")

	// custom method which tell if node has internet connection etc.
	// mse.IsConnected = func() bool {
	//	return true
	// }

	mse.Start()
	
	for {
		msg := doJob() 

		if  mse.IsMaster() {
			// do something 
			// MQ.publish(msg)
			// socket.write(msg)
			// etc.
		} else {
			// do nothing
		}
		log.Printf("%v", mse.IsMaster())
		time.Sleep(1 * time.Second)
	}
}
```

publish to mse channel "set-master [nodename]" to switch node to master

