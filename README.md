### Tested devices
- Huawei K3556

### Installation
Run:

    go get github.com/scr34m/gsmmodem

### Usage
Example:

```go
package main

import (
	"fmt"
	"github.com/scr34m/gsmmodem"
)

func main() {
	modem, err := gsmmodem.Open("/dev/ttyUSB3", 115200, "2250")
	if err != nil {
		panic(err)
	}

	list, err := modem.ListMessages()
	if err != nil {
		panic(err)
	}

	for _, msg := range *list {
		fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
		modem.DeleteMessage(msg.Index)
	}

	err = modem.ReaderMode()
	if err != nil {
		panic(err)
	}

	for packet := range modem.Receiver {
		switch p := packet.(type) {
		case gsmmodem.MessageNotification:
			msg, err := modem.GetMessage(p.Index)
			if err == nil {
				fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
				modem.DeleteMessage(p.Index)
			}
		}
	}
}
```