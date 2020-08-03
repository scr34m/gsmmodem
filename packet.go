package gsmmodem

import "time"

type Packet interface{}

type OK struct{}

type ERROR struct{}

type Unknown struct {
	Command string
	Value   string
}

// +GMM=...
type DeviceModelInformation struct {
	Value string
}

// +CPIN=...
type PinInformation struct {
	Value string
}

// +CGMI=...
type DeviceManufacturerInformation struct {
	Value string
}

// +CSCS=...
type CharacterSetInformation struct {
	Value string
}

// +CMTI=
type MessageNotification struct {
	Slot  string
	Index int
}

// +CMGR
type Message struct {
	Index     int
	Status    string
	Telephone string
	Timestamp time.Time
	Body      string
	Last      bool
}

// +CMGL
type MessageList []Message

// +CPMS=...
type StorageInfo struct {
	UsedSpace1, MaxSpace1 int
	UsedSpace2, MaxSpace2 int
	UsedSpace3, MaxSpace3 int
}
