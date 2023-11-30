package VClock

import (
	"VClockMQTT"
	"VClockMessageTypes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var wg sync.WaitGroup

type VClock struct {
	GroupName string
	MyName    string
	Clients   []VClockClient
	Mqtt      VClockMQTT.VClockMQTT
	Registerd bool
}

type VClockClient struct {
	Name    string
	Nthread int
}

func (vc *VClock) ReadGroupInfo() {
	f, e := os.ReadFile("./groupinfo.json")
	if e != nil {
		fmt.Println(e.Error())
		panic(e)
	}
	var ginfo VClockMessageTypes.GroupInfo
	json.Unmarshal(f, &ginfo)

	vc.GroupName = ginfo.GroupName
	for _, sim := range ginfo.Sims {
		vc.Clients = append(vc.Clients, VClockClient{sim.Name, 1})
	}

}

func (vc *VClock) ReadMyInfo() {
	f, e := os.ReadFile("./myinfo.json")
	if e != nil {
		fmt.Println(e.Error())
		panic(e)
	}
	var myinfo VClockMessageTypes.MyInfo
	json.Unmarshal(f, &myinfo)

	vc.MyName = myinfo.MyName
}

func (vc *VClock) Register(nthread int) {
	// client should have only one simulator info
	vc.Clients[0].Nthread = nthread

	// 起動するインスタンス数を送信する
	bytes, _ := json.Marshal(VClockMessageTypes.SimulatorInfo{
		Name:     vc.MyName,
		Nthreads: nthread,
	})
	vc.Mqtt.Publish(fmt.Sprintf("vclock/register/%s", vc.GroupName), bytes)
}

// TODO: 修了条件が時間の Delegate を追加
// TODO: 途中で強制終了のシーケンスを追加

func (vc *VClock) Delegate(fn func(), cond func() bool) {

	myInstanceID := -1
	localclock := 0
	token := -1

	// インスタンス登録に利用するトークンの生成
	rand.Seed(time.Now().UnixNano())
	token = rand.Intn(100)
	fmt.Printf("Generated Token: %d\n", token)

	/////////////////////// Subscribeのための関数群 ///////////////////////
	// 汎用の ACK ハンドラ
	var Ack MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		var t VClockMessageTypes.Ack
		json.Unmarshal(msg.Payload(), &t)

		fmt.Printf("[Ack] received on %s\n", t.Msg)
		vc.Registerd = true
	}

	// インスタンスIDの受信
	var recvAckInstantiate MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		var t VClockMessageTypes.AckInstantiate
		json.Unmarshal(msg.Payload(), &t)
		fmt.Printf("[AckInstance] received id: %d\n", t.Id)
		myInstanceID = t.Id
		wg.Done()
	}

	// 「実行開始（Run）」の受信
	var recvRun MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
		var t VClockMessageTypes.State
		json.Unmarshal(msg.Payload(), &t)
		fmt.Printf("[Run] received id: %d, clock: %d\n", t.Id, t.Clock)
		if myInstanceID == t.Id && localclock == t.Clock {
			wg.Done()
		} else {
			fmt.Printf("clock skew detected (global: %d, local: %d\n", t.Clock, localclock)
		}
	}

	//////////////////////////////////////////////////////////////////////

	// Ack をとりあえず受け取るようにする
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/ack/%s", vc.MyName), Ack)

	// インスタンスIDの受信ハンドラをSubscribe
	// TODO: Token をトピックに入れるかどうかを検討
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/ack/instantiate/%s/%s", vc.GroupName, vc.MyName), recvAckInstantiate)

	// インスタンスを vcserver に登録
	bytes, _ := json.Marshal(VClockMessageTypes.Instantiate{
		Name:  vc.MyName,
		Token: token,
	})

	// インスタンスIDの発行を要求
	vc.Mqtt.Publish(fmt.Sprintf("vclock/instantiate/%s/%s", vc.GroupName, vc.MyName), bytes)
	// インスタンスIDが発行されるまで待機
	wg.Add(1)
	wg.Wait()

	// 付与されたインスタンスIDを使って「実行開始（Run）」メッセージの受信ハンドラをSubscribe
	fmt.Printf("Instance ID: %d\n", myInstanceID)
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/run/%s/%s", vc.GroupName, vc.MyName), recvRun)

	state_send := func(clock int, state string) {
		bytes, _ := json.Marshal(VClockMessageTypes.State{
			Id:    myInstanceID,
			Clock: clock,
		})
		vc.Mqtt.Publish(fmt.Sprintf("vclock/%s/%s/%s", state, vc.GroupName, vc.MyName), bytes)
	}

	///////////////////////////// Delegate のメイン部分 /////////////////////////////
	for !cond() {
		/*
			// 途中で強制終了するかどうかを判定
			if receive(clock_final) {
				return
			}
		*/
		// vcserver に「実行準備完了（Ready）」を送信
		fmt.Printf("Ready %d\n", localclock)
		state_send(localclock, "ready")
		// vcserver から「実行開始」メッセージを受け取るまで待つ
		wg.Add(1)
		wg.Wait()
		// シミュレーションの１ステップ時間を実行
		fn()
		// シミュレーションの１ステップ「実行完了（Done）」を送信
		fmt.Printf("Done %d\n", localclock)
		state_send(localclock, "done")

		// ローカルクロックをインクリメントする
		localclock++
	}
	// 「実行終了（Complete）」を送信
	state_send(localclock, "complete")
}

func Initialize(mqttconfig string) VClock {
	vc := new(VClock)
	vc.Mqtt = VClockMQTT.Initialize(mqttconfig)

	vc.ReadGroupInfo()
	vc.ReadMyInfo()
	vc.Registerd = false
	return *vc
}
