package gsmmodem

import (
	"bufio"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/tarm/serial"
	"github.com/xlab/at/sms"
)

type Modem struct {
	port     io.ReadWriteCloser
	rx       chan Packet
	tx       chan string
	pin      string
	Receiver chan Packet
}

var OpenPort = func(config *serial.Config) (io.ReadWriteCloser, error) {
	return serial.OpenPort(config)
}

func Open(name string, baud int, pin string, debug bool) (*Modem, error) {
	port, err := OpenPort(&serial.Config{Name: name, Baud: baud})
	if debug {
		port = LogReadWriteCloser{port}
	}
	if err != nil {
		return nil, err
	}

	rx := make(chan Packet)
	tx := make(chan string)

	receiver := make(chan Packet, 16)

	modem := &Modem{
		port:     port,
		rx:       rx,
		tx:       tx,
		pin:      pin,
		Receiver: receiver,
	}

	// run send/receive goroutine
	go modem.listen()

	err = modem.init()
	if err != nil {
		return nil, err
	}
	return modem, nil
}

func lineChannel(r io.Reader) chan string {
	ret := make(chan string)
	go func() {
		buffer := bufio.NewReader(r)
		for {
			line, _ := buffer.ReadString(10)
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				continue
			}
			ret <- line
		}
	}()
	return ret
}

func isFinalStatus(status string) bool {
	return status == "OK" || status == "ERROR" || strings.Contains(status, "+CMS ERROR") || strings.Contains(status, "+CME ERROR")
}

var reQuestion = regexp.MustCompile(`AT(\+[A-Z]+)`)

func parsePacketBody(body string) ([]interface{}, bool) {
	ls := strings.SplitN(body, ":", 2)
	if len(ls) != 2 {
		return nil, true
	}
	uargs := strings.TrimSpace(ls[1])
	args := unquotes(uargs)
	return args, false
}

func parsePacket(header, body string) Packet {
	switch header {
	case "+GMM":
		return DeviceModelInformation{body}
	case "+CGMI":
		return DeviceManufacturerInformation{body}
	case "+CSCS":
		args, err := parsePacketBody(body)
		if err {
			break
		}
		return CharacterSetInformation{args[0].(string)}
	case "+CPIN":
		args, err := parsePacketBody(body)
		if err {
			break
		}
		return PinInformation{args[0].(string)}
	case "+CMTI":
		args, err := parsePacketBody(body)
		if err {
			break
		}
		return MessageNotification{args[0].(string), args[1].(int)}
	case "+CMGL":
		args, err := parsePacketBody(body)
		if err {
			break
		}

		log.Printf("Message: %s\n", args[4].(string))

		// TODO: TEXT type
		bs, err2 := hex.DecodeString(args[4].(string))
		if err2 != nil {
			panic(err)
		}
		msg := new(sms.Message)
		msg.ReadFrom(bs)
		return Message{Index: args[0].(int), Telephone: string(msg.Address), Timestamp: time.Time(msg.ServiceCenterTime), Body: msg.Text}
	case "+CMGR":
		args, err := parsePacketBody(body)
		if err {
			break
		}

		log.Printf("Message: %s\n", args[3].(string))

		// TODO: TEXT type
		bs, err2 := hex.DecodeString(args[3].(string))
		if err2 != nil {
			panic(err)
		}
		msg := new(sms.Message)
		msg.ReadFrom(bs)
		return Message{Index: args[0].(int), Telephone: string(msg.Address), Timestamp: time.Time(msg.ServiceCenterTime), Body: msg.Text}
	case "+CPMS":
		args, err := parsePacketBody(body)
		if err {
			break
		}
		if len(args) == 6 {
			return StorageInfo{
				args[0].(int), args[1].(int), args[2].(int), args[3].(int), args[4].(int), args[5].(int),
			}
		} else if len(args) == 4 {
			return StorageInfo{
				args[0].(int), args[1].(int), args[2].(int), args[3].(int), 0, 0,
			}
		} else if len(args) == 2 {
			return StorageInfo{
				args[0].(int), args[1].(int), 0, 0, 0, 0,
			}
		}
		break
	}
	return Unknown{header, body}
}

func (self *Modem) listen() {
	in := lineChannel(self.port)
	var echo, last, body string
	for {
		select {
		case line := <-in:
			if line == echo {
				continue // ignore echo of command
			}

			if strings.HasPrefix(line, "^") {
				continue // ignore ^BOOT, ^RSSI
			}

			if strings.HasPrefix(line, "+CMTI") {
				packet := parsePacket("+CMTI", line)
				self.Receiver <- packet
				continue
			}

			// final message (OK, ERROR) or we have a body but reading new command
			if isFinalStatus(line) || (body != "" && strings.HasPrefix(line, "+")) {
				packet := parsePacket(last, body)
				self.rx <- packet

				// XXX send OK packet to escape channel read in ListMessages()
				if isFinalStatus(line) && strings.HasPrefix(body, "+CMGL") {
					self.rx <- OK{}
				}

				body = ""

				if isFinalStatus(line) {
					continue
				}
			}

			if body == "" {
				body = line
			} else {
				body += "," + line
			}
		case line := <-self.tx:
			m := reQuestion.FindStringSubmatch(line)
			if len(m) > 0 {
				last = m[1]
			}
			echo = strings.TrimRight(line, "\r\n")
			self.port.Write([]byte(line))
		}
	}
}

func command(cmd string, quote bool, args ...interface{}) string {
	line := "AT" + cmd
	if len(args) > 0 {
		line += "=" + join(quote, args)
	}
	line += "\r\n"
	return line
}

func (self *Modem) send(cmd string, args ...interface{}) (Packet, error) {
	self.tx <- command(cmd, true, args...)
	response := <-self.rx
	if _, e := response.(ERROR); e {
		return response, errors.New("Response was ERROR")
	}
	return response, nil
}

func (self *Modem) sendRaw(cmd string, args ...interface{}) (Packet, error) {
	self.tx <- command(cmd, false, args...)
	response := <-self.rx
	if _, e := response.(ERROR); e {
		return response, errors.New("Response was ERROR")
	}
	return response, nil
}

// TODO error codes translation:
// +CME ERROR: 11 = SIM PIN required
// https://www.micromedia-int.com/en/gsm-2/73-gsm/669-cme-error-gsm-equipment-related-errors

func (self *Modem) init() error {
	// clear settings
	if _, err := self.send("Z"); err != nil {
		return err
	}
	log.Println("Reset")

	// turn off echo
	if _, err := self.send("E0"); err != nil {
		return err
	}
	log.Println("Echo off")

	// enable use of result code
	if _, err := self.send("+CMEE", "1"); err != nil {
		return err
	}
	log.Println("Enable use of result codes")

	msg, err := self.send("+GMM")
	if err != nil {
		return err
	}
	log.Printf("Device model: %s\n", msg.(DeviceModelInformation).Value)

	msg2, err := self.send("+CGMI")
	if err != nil {
		return err
	}
	log.Printf("Manufacturer: %s\n", msg2.(DeviceManufacturerInformation).Value)

	msgp, err := self.send("+CPIN?")
	if err != nil {
		return err
	}
	log.Printf("Pin status: %s\n", msgp.(PinInformation).Value)

	if msgp.(PinInformation).Value != "READY" {
		if _, err = self.send("+CPIN", self.pin); err != nil {
			return err
		}
	}

	msg3, err := self.send("+CSCS?")
	if err != nil {
		return err
	}
	log.Printf("Character set: %s\n", msg3.(CharacterSetInformation).Value)

	// storage SIM
	// ME, MT - flash storage
	// SM - SIM storage
	// SR - status report storage
	msg4, err := self.send("+CPMS", "SM", "SM", "SM")
	if err != nil {
		return err
	}
	log.Printf("Sim storage: %d/%d used\n", msg4.(StorageInfo).UsedSpace1, msg4.(StorageInfo).MaxSpace1)

	log.Printf("Initialisation completed\n")

	return nil
}

// ReadMode set on notification mode about incoming messages
func (self *Modem) ReaderMode() error {
	// forward new messages to the PC
	if _, err := self.sendRaw("+CNMI", "1,1"); err != nil {
		return err
	}
	log.Printf("Entered sms reader mode...\n")
	return nil
}

// ListMessages stored in memory.
func (self *Modem) ListMessages() (*MessageList, error) {
	// PUD: 0 unreaded, 1 readed, 4 all
	// TEXT: TODO
	packet, err := self.sendRaw("+CMGL", "4")
	if err != nil {
		return nil, err
	}
	res := MessageList{}
	if _, ok := packet.(OK); ok {
		return &res, nil
	}

	for {
		if msg, ok := packet.(Message); ok {
			res = append(res, msg)
		} else {
			break
		}
		packet = <-self.rx
	}
	return &res, nil
}

// GetMessage by index n from memory.
func (self *Modem) GetMessage(n int) (*Message, error) {
	packet, err := self.send("+CMGR", n)
	if err != nil {
		return nil, err
	}
	if msg, ok := packet.(Message); ok {
		return &msg, nil
	}
	return nil, errors.New("Message not found")
}

// DeleteMessage by index n from memory.
func (self *Modem) DeleteMessage(n int) error {
	_, err := self.send("+CMGD", n)
	return err
}
