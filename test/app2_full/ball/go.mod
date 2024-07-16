module ball

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
	github.com/go-fonts/liberation v0.3.0 // indirect
	github.com/go-latex/latex v0.0.0-20230307184459-12ec69307ad9 // indirect
	github.com/go-pdf/fpdf v0.6.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/rs/xid v1.4.0 // indirect
	golang.org/x/exp v0.0.0-20230307190834-24139beb5833 // indirect
	golang.org/x/image v0.6.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.22.0
	golang.org/x/text v0.8.0 // indirect
)
