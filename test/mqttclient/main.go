package main

//TODO: encapsulate MQTT import into VClockMQTT
import (
	"VClock"
	"VClockMessageTypes"
	"encoding/json"
	"fmt"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func Test(client MQTT.Client, msg MQTT.Message) {
	var t VClockMessageTypes.TestData
	json.Unmarshal(msg.Payload(), &t)

	fmt.Printf("Received message ID %s, Num %d\n", t.ID, t.Num)
}

type Params struct {
	param1 int
	param2 int
	result int
}

func (p *Params) Task() {
	p.result += p.param1 + p.param2
}

func (p *Params) Condition() bool {
	return p.result > 20
}

func main() {

	vc := VClock.Initialize("../../config/mqtt.conf")
	// シミュレータで利用するスレッド数の登録
	idList, err := vc.Register(1)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	// シミュレータで利用する入出力の用意
	p := Params{1, 1, 0}

	// シミュレーションタスクの委任
	// (シミュレーション関数, 終了条件)
	vc.Delegate(idList[0], p.Task, p.Condition)

	// シミュレーション結果の確認
	fmt.Printf("Simulation was done, %d\n", p.result)

	// 入力待ち
	fmt.Scanln()
	// MQTTブローカーから切断
	// （TODO: 後で VClock ライブラリに移す）
	vc.Mqtt.Close()
}
