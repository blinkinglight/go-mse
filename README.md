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
	mse.Start()
	for {
		if  mse.IsMaster() {
			// do something 
		}
		log.Printf("%v", mse.IsMaster())
		time.Sleep(1 * time.Second)
	}
}
```

