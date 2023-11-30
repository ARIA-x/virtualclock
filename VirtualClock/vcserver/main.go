package main

//TODO: encapsulate MQTT import into VClockMQTT
import (
	"VClock"
	"VClockMessageTypes"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type State int

const (
	None State = iota
	Ready
	Run
	Done
	Complete
)

type InstanceState struct {
	token int
	id    int
	clock int
	state State
}

type SimulatorState struct {
	name       string
	ninstances int
	activated  int
	istates    []InstanceState
}

type SimulationState struct {
	name    string
	sstates []SimulatorState
}

type WaitGroupCount struct {
	sync.WaitGroup
	count int64
}

// （全体）シミュレーションの状態保持変数
var simulationstate SimulationState

// 全インスタンスの実行同期用 WaitGroup
// var wg sync.WaitGroup
var wgc WaitGroupCount

// グローバル仮想時刻
var globalclock int

func (wg *WaitGroupCount) Add(delta int) {
	atomic.AddInt64(&wg.count, int64(delta))
	wg.WaitGroup.Add(delta)
}

func (wg *WaitGroupCount) Done() {
	atomic.AddInt64(&wg.count, -1)
	wg.WaitGroup.Done()
}

func (wg *WaitGroupCount) GetCount() int {
	return int(atomic.LoadInt64(&wg.count))
}

func Test(client MQTT.Client, msg MQTT.Message) {

	var t VClockMessageTypes.TestData
	json.Unmarshal(msg.Payload(), &t)

	fmt.Printf("Received message ID %s, Num %d\n", t.ID, t.Num)

	// ack to test client
	bytes, _ := json.Marshal(VClockMessageTypes.TestData{
		ID:  "from server",
		Num: t.Num + 1,
	})
	token := client.Publish("test/client", 0, false, bytes)
	token.Wait()
}

/////////////////////// Subscribeのための関数群 ///////////////////////

// インスタンス数登録ハンドラ
func recvRegister(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.SimulatorInfo
	json.Unmarshal(msg.Payload(), &data)

	for i, ss := range simulationstate.sstates {
		if ss.name == data.Name {
			fmt.Printf("[Register] Message received from...\n")
			fmt.Printf("\tName: %s\n", data.Name)
			fmt.Printf("\tNumber of threads: %d\n", data.Nthreads)

			//インスタンスの数を記録
			simulationstate.sstates[i].ninstances = data.Nthreads
		}
	}

	// ack を返す
	// (ここだけ vc を使って返せない…後で要検討)
	bytes, _ := json.Marshal(VClockMessageTypes.Ack{
		Msg: "Register",
	})
	topic := fmt.Sprintf("vclock/ack/%s", data.Name)
	if token := client.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}

func recvInstantiate(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.Instantiate
	json.Unmarshal(msg.Payload(), &data)

	for i, ss := range simulationstate.sstates {

		if ss.activated == ss.ninstances {
			fmt.Println("  [Warning]!!! all instances have already activated")
			break
		}

		if ss.name == data.Name {
			var is InstanceState
			is.token = data.Token
			is.state = None
			is.clock = 0
			is.id = ss.activated
			simulationstate.sstates[i].istates = append(simulationstate.sstates[i].istates, is)

			// activate されたインスタンスを一つ増やす
			simulationstate.sstates[i].activated += 1

			//インスタンスIDを返信する
			bytes, _ := json.Marshal(VClockMessageTypes.AckInstantiate{
				Token: data.Token,
				Id:    is.id,
			})
			//TODO: need atomic access to "activated" variable
			topic := fmt.Sprintf("vclock/ack/instantiate/%s/%s", simulationstate.name, ss.name)
			if token := client.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		}
	}

	// メイン関数を起動して、申請のあった
	// 全インスタンスからID申請があったか確認する
	wgc.Done()
}

func recvReady(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &data)

	arr := strings.Split(msg.Topic(), "/")
	sim_name := arr[len(arr)-1]

	// インスタンスの実行状態を Ready に変更
	for i, ss := range simulationstate.sstates {
		if ss.name == sim_name {
			simulationstate.sstates[i].istates[data.Id].clock = data.Clock
			simulationstate.sstates[i].istates[data.Id].state = Ready
		}
	}
	// メイン関数を起動して
	// 全インスタンスが実行準備完了か確認する
	if wgc.GetCount() > 0 && data.Clock == globalclock {
		wgc.Done()
	}
}

func recvDone(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &data)

	arr := strings.Split(msg.Topic(), "/")
	sim_name := arr[len(arr)-1]

	// インスタンスの実行状態を Done に変更
	for i, ss := range simulationstate.sstates {
		if ss.name == sim_name {
			fmt.Printf("%s, %s\n", ss.name, sim_name)
			simulationstate.sstates[i].istates[data.Id].clock = data.Clock
			simulationstate.sstates[i].istates[data.Id].state = Done
		}
	}
	// メイン関数を起動して
	// 全インスタンスが実行完了か確認する
	if wgc.GetCount() > 0 && data.Clock == globalclock {
		wgc.Done()
	}
}

func recvComplete(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &data)

	arr := strings.Split(msg.Topic(), "/")
	sim_name := arr[len(arr)-1]

	// インスタンスの実行状態を Complete に変更
	for i, ss := range simulationstate.sstates {
		if ss.name == sim_name {
			simulationstate.sstates[i].istates[data.Id].clock = data.Clock
			simulationstate.sstates[i].istates[data.Id].state = Complete
		}
	}

}

//////////////////////////////////////////////////////////////////////

func main() {

	// 仮想時刻機能の初期化
	vc := VClock.Initialize("../../config/mqtt.conf")

	//（全体）シミュレーションデータの初期化
	simulationstate.name = vc.GroupName
	simulationstate.sstates = make([]SimulatorState, len(vc.Clients))
	//（個別）シミュレータデータの初期化
	for i, cli := range vc.Clients {
		simulationstate.sstates[i].name = cli.Name
		simulationstate.sstates[i].activated = 0
		simulationstate.sstates[i].ninstances = 0
	}

	// グローバル時刻の初期化
	globalclock = 0

	// インスタンス数登録のSubscribe
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/register/%s", vc.GroupName), recvRegister)

	// インスタンス登録のSubscribe
	for _, cli := range vc.Clients {
		vc.Mqtt.Subscribe(fmt.Sprintf("vclock/instantiate/%s/%s", vc.GroupName, cli.Name), recvInstantiate)
	}

	// 実行準備完了（Ready）のSubscribe
	for _, cli := range vc.Clients {
		vc.Mqtt.Subscribe(fmt.Sprintf("vclock/ready/%s/%s", vc.GroupName, cli.Name), recvReady)
	}

	// 実行完了（Done）のSubscribe
	for _, cli := range vc.Clients {
		vc.Mqtt.Subscribe(fmt.Sprintf("vclock/done/%s/%s", vc.GroupName, cli.Name), recvDone)
	}

	// 実行終了（Complete）のSubscribe
	for _, cli := range vc.Clients {
		vc.Mqtt.Subscribe(fmt.Sprintf("vclock/complete/%s/%s", vc.GroupName, cli.Name), recvComplete)
	}

	ready_to_run := func() bool {
		for _, ss := range simulationstate.sstates {
			if ss.activated != ss.ninstances {
				return false
			}
		}
		return true
	}

	///////// シミュレーション準備サイクル /////////
	// すべてのシミュレータのインスタンスが起動したかどうか確認
	for {
		wgc.Add(1)
		wgc.Wait()
		if ready_to_run() {
			fmt.Println("vcserver received instantiate requests from all instances")
			break
		}
	}

	state_check := func(state State) bool {
		for _, ss := range simulationstate.sstates {
			for _, is := range ss.istates {
				switch state {
				case Complete:
					if is.state != state {
						return false
					}
				default:
					if is.clock <= globalclock && is.state != state {
						return false
					}
				}
			}
		}
		return true
	}

	///////// シミュレーション実行サイクル /////////
	for {
		// すべてのシミュレータのインスタンスの実行が完了したらシミュレーションを終了する
		if state_check(Complete) {
			break
		}

		// 実行準備完了を確認
		for {
			if state_check(Ready) {
				break
			} else {
				wgc.Add(1)
				wgc.Wait()
			}
		}
		// １シミュレーションサイクルを実行
		for _, ss := range simulationstate.sstates {
			for _, is := range ss.istates {
				bytes, _ := json.Marshal(VClockMessageTypes.State{
					Id:    is.id,
					Clock: globalclock,
				})
				vc.Mqtt.Publish(fmt.Sprintf("vclock/run/%s/%s", vc.GroupName, ss.name), bytes)
			}
		}

		// 実行完了を確認
		for {
			if state_check(Done) {
				break
			} else {
				wgc.Add(1)
				wgc.Wait()
			}
		}
		// 次のシミュレーションサイクルに移行
		globalclock++
	}

	fmt.Println("Simulation Completed")
	// 入力待ち
	fmt.Scanln()
	// MQTTブローカーから切断
	vc.Mqtt.Close()
}

/*
メモ
・グループID、参加するアプリケーション名は、CoupledSimulationがファイルに落とす
　落としたファイルを Pod 作成時に指定フォルダに置いておくという体で
　今回は、それぞれのフォルダに入れておく
  （クライアントは自分の名前だけ、サーバは全員の名前を把握）
・mqtt の設定を groupinfo.json に入れるかどうか後々検討する
　（入れると mqtt.conf を読み込まなくていいけど、散らばった json の更新が面倒）

  generics
  https://zenn.dev/nobishii/articles/type_param_intro
*/
