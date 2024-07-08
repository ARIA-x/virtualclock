module player

go 1.19

replace VirtualClock => ../../../VirtualClock

replace VClock => ../../../VirtualClock/libs/Vclock

replace VClockMQTT => ../../../VirtualClock/libs/VclockMQTT

replace VClockMessageTypes => ../../../VirtualClock/libs/VclockMessageTypes

replace VClockDataTypes => ../../../VirtualClock/libs/VclockDataTypes

require (
	VClock v0.0.0-00010101000000-000000000000
	gonum.org/v1/plot v0.12.0
)

require (
	VClockDataTypes v0.0.0-00010101000000-000000000000 // indirect
	VClockMQTT v0.0.0-00010101000000-000000000000 // indirect
	VClockMessageTypes v0.0.0-00010101000000-000000000000 // indirect
	git.sr.ht/~sbinet/gg v0.3.1 // indirect
	github.com/ajstarks/svgo v0.0.0-20211024235047-1546f124cd8b // indirect
	github.com/eclipse/paho.mqtt.golang v1.4.2 // indirect
	github.com/go-fonts/liberation v0.2.0 // indirect
	github.com/go-latex/latex v0.0.0-20210823091927-c0d11ff05a81 // indirect
	github.com/go-pdf/fpdf v0.6.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/rs/xid v1.4.0 // indirect
	golang.org/x/image v0.0.0-20220902085622-e7cb96979f69 // indirect
	golang.org/x/net v0.0.0-20201021035429-f5854403a974 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/text v0.3.7 // indirect
)
