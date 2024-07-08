module vcserver

go 1.19


replace VClockMQTT => ../libs/VclockMQTT

replace VClockMessageTypes => ../libs/VClockMessageTypes

replace VClockDataTypes => ../libs/VClockDataTypes

require VClockMQTT v0.0.0-00010101000000-000000000000

require (
	VClockDataTypes v0.0.0-00010101000000-000000000000
	VClockMessageTypes v0.0.0-00010101000000-000000000000
)

require golang.org/x/net v0.2.0 // indirect

require (
	github.com/eclipse/paho.mqtt.golang v1.4.2
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/rs/xid v1.4.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
)
