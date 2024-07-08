module VClock

go 1.19

replace VClockMQTT => ../VClockMQTT

replace VClockMessageTypes => ../VClockMessageTypes

replace VClockDataTypes => ../VClockDataTypes

require (
	VClockDataTypes v0.0.0-00010101000000-000000000000
	VClockMQTT v0.0.0-00010101000000-000000000000
	VClockMessageTypes v0.0.0-00010101000000-000000000000
)

require (
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/rs/xid v1.4.0 // indirect
	golang.org/x/net v0.0.0-20200425230154-ff2c4b7c35a0 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
)
