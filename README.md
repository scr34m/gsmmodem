## Go library for GSM modems

A Go library for the receiving SMS messages through a GSM modem.
Rewrite / use of https://github.com/barnybug/gogsmmodem

### Tested devices
- Huawei K3556
- Huawei E153

### Notes

The notifications caused by `AT+CNMI` are not always sent back on the first USB serial you have to check each serial ports which one is working for you. Usally the highest numbered serial port is the good one.

On weak 3G signal with Huawei E153 maybe better to switch GSM only mode:
```
AT^HSDPA=0
AT^SYSCFG=13,0,3FFFFFFF,0,0
```

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
	modem, err := gsmmodem.Open("/dev/ttyUSB3", 115200, "2250", true)
	if err != nil {
		panic(err)
	}

	// Read messages stored on the SIM
	list, err := modem.ListMessages()
	if err != nil {
		panic(err)
	}

	for _, msg := range *list {
		fmt.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
		modem.DeleteMessage(msg.Index)
	}

	// Switch to receiver mode so notifications received on every new message
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
