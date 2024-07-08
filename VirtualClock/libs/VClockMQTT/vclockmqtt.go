package VClockMQTT

import (
	"fmt"
	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/xid"
)

type VClockMQTT struct {
	address string
	client  MQTT.Client
	opts    *MQTT.ClientOptions
}

func Initialize(configFile string) VClockMQTT {
	vcmqtt := new(VClockMQTT)

	// broker アドレスの読み込み
	bytes, err := os.ReadFile(configFile)
	if err != nil {
		panic(err)
	}
	vcmqtt.address = string(bytes)

	vcmqtt.opts = MQTT.NewClientOptions().AddBroker(vcmqtt.address).SetClientID(xid.New().String())
	vcmqtt.opts.OnConnect = func(client MQTT.Client) {
		fmt.Printf("Connected to a MQTT Broker\n")
	}

	// broker に接続
	vcmqtt.client = MQTT.NewClient(vcmqtt.opts)
	if token := vcmqtt.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return *vcmqtt
}

func (vcmqtt *VClockMQTT) Subscribe(topic string, handler MQTT.MessageHandler) {
	if token := vcmqtt.client.Subscribe(topic, 0, handler); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func (vcmqtt *VClockMQTT) Publish(topic string, payload []byte) {
	if token := vcmqtt.client.Publish(topic, 0, false, payload); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func (vcmqtt *VClockMQTT) Close() {
	// MQTTブローカーから切断
	vcmqtt.client.Disconnect(250)
	fmt.Println("MQTT Disconnected")
}
