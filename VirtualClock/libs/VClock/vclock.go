package VClock

import (
	"VClockDataTypes"
	"VClockMQTT"
	"VClockMessageTypes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

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

type VClock struct {
	GroupName  string
	MyName     string
	Mqtt       VClockMQTT.VClockMQTT
	MqttConfig string
	Instances  []VClockInstance
	Wg         sync.WaitGroup
	ClockCycle int
}

type VClockInstance struct {
	InstanceID int
	LocalClock int
	Token      int
	Mqtt       VClockMQTT.VClockMQTT
	Wg         sync.WaitGroup
	Vc         *VClock
	State      State
	ClockCycle int
	NextClock  int
}

func (vc *VClock) ReadConfiguration(filename string) {
	f, e := os.ReadFile(filename)
	if e != nil {
		fmt.Println(e.Error())
		panic(e)
	}
	var conf VClockDataTypes.Configuration
	json.Unmarshal(f, &conf)

	// TODO: エラー処理
	vc.GroupName = conf.GroupName
	vc.MyName = conf.Name
	vc.ClockCycle = conf.ClockCycle
}

// スレッド数の登録関数
// 引数：実行スレッド数
// 返値：vcserver から割り当てられたインスタンスIDリスト
func (vc *VClock) Register(nthread int) ([]int, error) {
	if nthread <= 0 {
		return nil, errors.New("the number of threads must be more than one")
	}

	// インスタンス構造体の初期化
	for i := 0; i < nthread; i++ {
		vc.Instances = append(vc.Instances, VClockInstance{
			InstanceID: -1,
			LocalClock: 0,
			Token:      -1, //将来的に削除予定
			Wg:         sync.WaitGroup{},
			Vc:         vc,
			State:      None,
			ClockCycle: vc.ClockCycle,
			NextClock:  0,
		})
	}

	// 起動するインスタンス数を送信する
	bytes, _ := json.Marshal(VClockMessageTypes.Register{
		Name:  vc.MyName,
		Num:   nthread,
		Cycle: vc.ClockCycle,
	})
	vc.Mqtt.Publish(fmt.Sprintf("vclock/register/%s", vc.GroupName), bytes)

	// Registerが終了するまで待機
	// TODO:タイムアウト処理
	vc.Wg.Add(1)
	vc.Wg.Wait()

	/* Register と Instantiate プロトコルを将来的に分ける可能性があるので残しておく
	// インスタンスの構造体を作成
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < nthread; i++ {
		vc.Instances = append(vc.Instances, VClockInstance{
			InstanceID: -1,
			LocalClock: 0,
			Token:      rand.Intn(100),
			Wg:         sync.WaitGroup{},
			Vc:         vc,
		})
	}

	// インスタンスを vcserver に登録
	var data []int
	for i := range vc.Instances {
		data = append(data, vc.Instances[i].Token)
	}
	bytes, _ = json.Marshal(VClockMessageTypes.Instantiate{
		Name:  vc.MyName,
		Token: data,
	})

	// インスタンスIDの発行を要求
	vc.Mqtt.Publish(fmt.Sprintf("vclock/instantiate/%s/%s", vc.GroupName, vc.MyName), bytes)

	// インスタンスIDが発行されるまで待機
	// TODO:タイムアウト処理
	vc.Wg.Add(1)
	vc.Wg.Wait()
	*/

	// 取得したインスタンスIDリストを返す
	var ret []int
	for i := range vc.Instances {
		ret = append(ret, vc.Instances[i].InstanceID)
		fmt.Printf("Simulator (%s) acquired InstanceID %d\n", vc.MyName, vc.Instances[i].InstanceID)
	}

	return ret, nil
}

/////////////////////// Subscribeのための関数群 ///////////////////////

func (vc *VClock) ack(client MQTT.Client, msg MQTT.Message) {
	var t VClockMessageTypes.Ack
	json.Unmarshal(msg.Payload(), &t)
	fmt.Printf("[Ack] received %s\n", t.Msg)
	vc.Wg.Done()
}

// インスタンスIDの受信
func (vc *VClock) recvAckInstantiate(client MQTT.Client, msg MQTT.Message) {
	var t VClockMessageTypes.AckInstantiate
	json.Unmarshal(msg.Payload(), &t)
	for _, registeredInstance := range t.Instances {
		fmt.Printf("\t[AckInstantiate] received id: %d\n", registeredInstance.Id)
		//TODO: use generics
		for i := range vc.Instances {
			if vc.Instances[i].Token == registeredInstance.Token {
				vc.Instances[i].InstanceID = registeredInstance.Id
			}
		}

	}
	vc.Wg.Done()
}

// インスタンス登録完了・インスタンスIDの受信
func (vc *VClock) recvAckRegister(client MQTT.Client, msg MQTT.Message) {
	var t VClockMessageTypes.AckRegister
	json.Unmarshal(msg.Payload(), &t)

	// インスタンスIDの設定
	for i := range vc.Instances {
		fmt.Printf("\t[AckInstantiate] received id: %d\n", t.ID[i])
		vc.Instances[i].InstanceID = t.ID[i]
	}
	// MQTT 接続
	for i := range vc.Instances {
		vc.Instances[i].Mqtt = VClockMQTT.Initialize(vc.MqttConfig)
	}
	vc.Wg.Done()
}

// 「実行開始（Run）」の受信
func (vci *VClockInstance) recvRun(client MQTT.Client, msg MQTT.Message) {
	// シミュレーションを終了している場合や、同期対象外の時刻であればスキップ
	if vci.State == Complete {
		return
	}
	var t VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &t)
	fmt.Printf("[Run] (ID: %d) received clock: %d\n", vci.InstanceID, t.Clock)
	if vci.LocalClock == t.Clock {
		vci.Wg.Done()
	} else {
		fmt.Printf("clock skew detected on instance [%d] (%d) (global: %d, local: %d)\n", vci.InstanceID, t.Id, t.Clock, vci.LocalClock)
	}
}

// 「次のステップ（Next）」の受信
func (vci *VClockInstance) recvNext(client MQTT.Client, msg MQTT.Message) {
	// シミュレーションを終了している場合や、同期対象外の時刻であればスキップ
	if vci.State == Complete {
		return
	}
	var t VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &t)
	fmt.Printf("[Next] (ID: %d) received clock: %d\n", vci.InstanceID, t.Clock)
	if vci.LocalClock == t.Clock {

		// バグ回避の一時的な処置（後で必ず直す！）
		time.Sleep(50 * time.Millisecond) // 0.5秒待つときはMillisecondと乗算する
		vci.Wg.Done()
	} else {
		fmt.Printf("clock skew detected on instance [%d] (%d) (global: %d, local: %d)\n", vci.InstanceID, t.Id, t.Clock, vci.LocalClock)
	}
}

// ここまで（Subscribeのための関数群）
//////////////////////////////////////////////////////////////////////

// VClock の初期化
func Initialize(mqttconfig string) *VClock {
	vc := new(VClock)
	vc.Mqtt = VClockMQTT.Initialize(mqttconfig)
	vc.MqttConfig = mqttconfig

	vc.ReadConfiguration("./configuration.json")
	// Ack をとりあえず受け取るようにする
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/ack/%s", vc.MyName), vc.ack)
	// インスタンスIDの受信ハンドラをSubscribe
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/ack/register/%s/%s", vc.GroupName, vc.MyName), vc.recvAckRegister)
	// インスタンスIDの受信ハンドラをSubscribe
	vc.Mqtt.Subscribe(fmt.Sprintf("vclock/ack/instantiate/%s/%s", vc.GroupName, vc.MyName), vc.recvAckInstantiate)
	return vc
}

// 状態送信関数
func (vci *VClockInstance) state_send(clock int, state string) {
	bytes, _ := json.Marshal(VClockMessageTypes.State{
		Id:    vci.InstanceID,
		Clock: clock,
	})
	vc := vci.Vc
	vci.Mqtt.Publish(fmt.Sprintf("vclock/%s/%s/%s", state, vc.GroupName, vc.MyName), bytes)
	fmt.Printf("-- [ID: %d] State Sent %d :%s\n", vci.InstanceID, vci.LocalClock, state)

}

func (vc *VClock) searchVClockInstance(instanceID int) (*VClockInstance, error) {
	for i := range vc.Instances {
		if vc.Instances[i].InstanceID == instanceID {
			return &vc.Instances[i], nil
		}
	}
	return &VClockInstance{}, fmt.Errorf("could not find an instance [%d]", instanceID)
}

// 周期タスクの完全委託
// TODO: localclock の上限による途中終了の検討
// TODO: 途中で強制終了のシーケンスを追加
func (vc *VClock) Delegate(instanceID int, fn func(), cond func() bool) {
	vci, err := vc.searchVClockInstance(instanceID)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	//vci.Mqtt = VClockMQTT.Initialize("../../config/mqtt.conf")

	// 実行開始（Run）メッセージの受信ハンドラをSubscribe
	vci.Mqtt.Subscribe(fmt.Sprintf("vclock/run/%s/%s", vc.GroupName, vc.MyName), vci.recvRun)
	// 次のステップ（Next）メッセージの受信ハンドラをSubscribe
	vci.Mqtt.Subscribe(fmt.Sprintf("vclock/next/%s/%s", vc.GroupName, vc.MyName), vci.recvNext)
	for !cond() {
		vci.State = Ready
		// vcserver に「実行準備完了（Ready）」を送信
		vci.state_send(vci.LocalClock, "ready")
		// vcserver から「実行開始」メッセージを受け取るまで待つ
		vci.Wg.Add(1)
		vci.Wg.Wait()

		// 次の実行時刻でなければスキップ
		if vci.LocalClock != vci.NextClock {
			fmt.Printf("[skipped] simulation clock %d was skipped\n", vci.LocalClock)
			// vcserver から「次のステップ」メッセージを受け取るまで待つ
			vci.Wg.Add(1)
			vci.Wg.Wait()
			vci.LocalClock++
			continue
		}
		// シミュレーションの１ステップ時間を実行
		fn()

		// シミュレーションの１ステップ「実行完了（Done）」を送信
		vci.State = Done
		vci.state_send(vci.LocalClock, "done")
		// vcserver から「次のステップ」メッセージを受け取るまで待つ
		vci.Wg.Add(1)
		vci.Wg.Wait()
		// 次のサイクル実行時刻を設定
		vci.NextClock += vci.ClockCycle
		// ローカルクロックをインクリメントする
		vci.LocalClock++
	}
	//シミュレーション完了（complete）を通知
	vci.State = Complete
	vci.state_send(vci.LocalClock, "complete")
}

// 周期タスクの部分指定（ステップ実行開始）
func (vc *VClock) StepBegin(instanceID int) {
	vci, err := vc.searchVClockInstance(instanceID)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	// 実行開始（Run）メッセージの受信ハンドラをSubscribe
	vci.Mqtt.Subscribe(fmt.Sprintf("vclock/run/%s/%s", vc.GroupName, vc.MyName), vci.recvRun)
	// 次のステップ（Next）メッセージの受信ハンドラをSubscribe
	vci.Mqtt.Subscribe(fmt.Sprintf("vclock/next/%s/%s", vc.GroupName, vc.MyName), vci.recvNext)

	for {
		// vcserver に「実行準備完了（Ready）」を送信
		vci.State = Ready
		vci.state_send(vci.LocalClock, "ready")
		// vcserver から「実行開始」メッセージを受け取るまで待つ
		vci.Wg.Add(1)
		vci.Wg.Wait()
		// 次の実行時刻でなければスキップ
		if vci.LocalClock != vci.NextClock {
			fmt.Printf("[skipped] simulation clock %d was skipped\n", vci.LocalClock)
			// vcserver から「次のステップ」メッセージを受け取るまで待つ
			vci.Wg.Add(1)
			vci.Wg.Wait()
			vci.LocalClock++
			continue
		} else {
			break
		}
	}
}

// 周期タスクの部分指定（ステップ実行終了）
func (vc *VClock) StepEnd(instanceID int) {
	vci, err := vc.searchVClockInstance(instanceID)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	// シミュレーションの１ステップ「実行完了（Done）」を送信
	vci.State = Done
	vci.state_send(vci.LocalClock, "done")
	// vcserver から「次のステップ」メッセージを受け取るまで待つ
	vci.Wg.Add(1)
	vci.Wg.Wait()
	// 次のサイクル実行時刻を設定
	vci.NextClock += vci.ClockCycle
	// ローカルクロックをインクリメントする
	vci.LocalClock++
}

// 周期タスクを実行する全ルーチンの終了を通知
// （親ルーチンから呼び出されることを想定）
func (vc *VClock) CompleteAll() {
	for i := range vc.Instances {
		// 「実行終了（Complete）」を送信
		vc.Instances[i].State = Complete
		vc.Instances[i].state_send(vc.Instances[i].LocalClock, "complete")
	}
}

// 周期タスクの部分指定（シミュレーション終了）
func (vc *VClock) Complete(instanceID int) {
	vci, err := vc.searchVClockInstance(instanceID)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	// 「実行終了（Complete）」を送信
	vci.State = Complete
	vci.state_send(vci.LocalClock, "complete")
}
