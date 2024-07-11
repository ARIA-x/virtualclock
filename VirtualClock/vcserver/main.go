package main

//TODO: encapsulate MQTT import into VClockMQTT
import (
	"VClockDataTypes"
	"VClockMQTT"
	"VClockMessageTypes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

// シミュレーションの状態保持変数
var simulation Simulation

// 全インスタンスの実行同期用 WaitGroup
var cwg CountableWaitGroup

// グローバル仮想時刻
var globalclock int

// 追加
var (
	logFile   *os.File
	csvWriter *csv.Writer
	logMutex  sync.Mutex
	startTime time.Time
)

func initLogger() error {
	var err error
	logFile, err = os.OpenFile("simulation_log.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	csvWriter = csv.NewWriter(logFile)

	// ファイルが新規作成された場合にのみヘッダーを書き込む
	fileInfo, err := logFile.Stat()
	if err != nil {
		return err
	}
	if fileInfo.Size() == 0 {
		// ヘッダーを書き込む
		err = csvWriter.Write([]string{
			"Timestamp",
			"ElapsedSeconds",
			"Type",
			"Name",
			"ID",
			"Clock",
			"State",
		})
		if err != nil {
			return err
		}
		csvWriter.Flush()
	}

	startTime = time.Now() // グローバル変数 startTime を初期化
	return nil
}

func logEvent(eventType, name string, id, clock int, state string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05.000000")
	elapsedSeconds := now.Sub(startTime).Seconds()

	err := csvWriter.Write([]string{
		timestamp,
		fmt.Sprintf("%.6f", elapsedSeconds), // 経過秒数（小数点以下6桁）
		eventType,
		name,
		fmt.Sprintf("%d", id),
		fmt.Sprintf("%d", clock),
		state,
	})
	if err != nil {
		fmt.Printf("Error writing to log: %v\n", err)
	}
	csvWriter.Flush()
}

func closeLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// MQ メッセージテスト用
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

/////////////////////// Subscribeのための関数群（ここから） ///////////////////////

// インスタンス数登録ハンドラ
func recvRegister(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.Register
	json.Unmarshal(msg.Payload(), &data)

	simulator, err := simulation.searchSimulator(data.Name)
	if err != nil {
		fmt.Print(err.Error())
	} else {
		fmt.Printf("[Register] Message received from...\n")
		fmt.Printf("\tName: %s\n", data.Name)
		fmt.Printf("\tNumber of threads: %d\n", data.Num)

		for i := 0; i < data.Num; i++ {
			// インスタンスの入れ物を作っておく
			instance := Instance{token: -1, id: simulation.GenerateID(), clock: -1, state: None, cycle: data.Cycle, nextclock: 0}
			simulator.instances = append(simulator.instances, instance)
		}

		// 生成したインスタンスのIDを返信
		var ids []int
		for i := range simulator.instances {
			ids = append(ids, simulator.instances[i].id)
		}

		bytes, _ := json.Marshal(VClockMessageTypes.AckRegister{
			Name: simulator.name,
			ID:   ids,
		})
		topic := fmt.Sprintf("vclock/ack/register/%s/%s", simulation.name, simulator.name)
		if token := client.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

		/*
			bytes, _ := json.Marshal(VClockMessageTypes.Ack{
				Msg: "Register",
			})
			topic := fmt.Sprintf("vclock/ack/%s", data.Name)
			if token := client.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
				panic(token.Error())
			}
		*/
	}
}

// インスタンス登録ハンドラ
// (現在利用されていません。インスタンス毎に ClockCycle を変えるときに利用するかも)
func recvInstantiate(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.Instantiate
	json.Unmarshal(msg.Payload(), &data)
	simulator, err := simulation.searchSimulator(data.Name)
	if err != nil {
		fmt.Print(err.Error())
		return
	}
	if simulator.isActivatedAll() {
		fmt.Println("\t[Warning]!!! all instances have already activated")
		return
	}

	// 事前に申請のあったインスタンス数と同じだけのトークンが送られてきた場合のみインスタンス生成
	// （※その他の場合は要検討 -> TODO）
	if len(data.Token) == len(simulator.instances) {
		for _, token := range data.Token {
			_, err := simulator.Activate(token)
			if err != nil {
				fmt.Print(err.Error())
				return
			}
		}

		//インスタンスIDを返信する
		var registered []VClockMessageTypes.RegisteredInstance
		for _, i := range simulator.instances {
			registered = append(registered, VClockMessageTypes.RegisteredInstance{
				Token: i.token,
				Id:    i.id,
			})
		}
		bytes, _ := json.Marshal(VClockMessageTypes.AckInstantiate{
			Instances: registered,
		})
		topic := fmt.Sprintf("vclock/ack/instantiate/%s/%s", simulation.name, simulator.name)
		if token := client.Publish(topic, 0, false, bytes); token.Wait() && token.Error() != nil {
			panic(token.Error())
		}

	} else {
		fmt.Printf("\t[Warning]!!! the requested number of instance is different from the registered number\n")
		fmt.Printf("\treceived [%d], registered[%d]\n", len(data.Token), len(simulator.instances))
		return
	}
	// メイン関数を起動して、申請のあった
	// 全インスタンスからID申請があったか確認する
	cwg.Done()
}

// Ready状態の受信ハンドラ
func recvReady(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &data)
	arr := strings.Split(msg.Topic(), "/")
	simulatorName := arr[len(arr)-1]

	simulation.StateTransition(simulatorName, data.Id, data.Clock, Ready)
	logEvent("Simulator", simulatorName, data.Id, data.Clock, "Ready")

	// メイン関数を起動して
	// 全インスタンスが実行準備完了か確認する
	if cwg.GetCount() > 0 && data.Clock == globalclock {
		cwg.Done()
	}
}

// Done状態の受信ハンドラ
func recvDone(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &data)
	arr := strings.Split(msg.Topic(), "/")
	simulatorName := arr[len(arr)-1]

	simulation.StateTransition(simulatorName, data.Id, data.Clock, Done)
	logEvent("Simulator", simulatorName, data.Id, data.Clock, "Done")

	// メイン関数を起動して
	// 全インスタンスが実行完了か確認する
	if cwg.GetCount() > 0 && data.Clock == globalclock {
		cwg.Done()
	}
}

// Complete状態の受信ハンドラ
func recvComplete(client MQTT.Client, msg MQTT.Message) {
	var data VClockMessageTypes.State
	json.Unmarshal(msg.Payload(), &data)
	arr := strings.Split(msg.Topic(), "/")
	simulatorName := arr[len(arr)-1]

	fmt.Printf("[Complete] Received from %d (t=%d)\n", data.Id, data.Clock)
	simulation.StateTransition(simulatorName, data.Id, data.Clock, Complete)
	logEvent("Simulator", simulatorName, data.Id, data.Clock, "Complete")

	if simulation.isCompleteAll() {
		// すべてのシミュレータのインスタンスの実行が完了したらシミュレーションを終了する
		fmt.Printf("Simulation complete at clock %d --------- \n", globalclock)
		// MQTTブローカーから切断
		// TODO: appropriate exit.
		// mqtt.Close()
		os.Exit(0)
	}
}

/////////////////////// Subscribeのための関数群（ここまで） ///////////////////////

func ReadSimulationStructure(filename string) VClockDataTypes.SimulationStructure {
	f, e := os.ReadFile(filename)
	if e != nil {
		fmt.Println(e.Error())
		panic(e)
	}
	var info VClockDataTypes.SimulationStructure
	json.Unmarshal(f, &info)
	return info
}

func main() {

	// 追加
	err := initLogger()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		return
	}
	defer closeLogger()

	// 仮想時刻機能の初期化
	mqtt := VClockMQTT.Initialize("../../config/mqtt.conf")

	// グループ情報の取得
	simulationstructure := ReadSimulationStructure("./structure.json")

	// シミュレーションデータの初期化
	simulation.name = simulationstructure.GroupName
	simulation.idseed = 0
	simulation.simulators = make([]Simulator, len(simulationstructure.Sims))

	// シミュレータデータの初期化
	for i, sim := range simulationstructure.Sims {
		simulation.simulators[i].name = sim.Name
	}

	// グローバル時刻の初期化
	globalclock = 0

	// インスタンス数登録のSubscribe
	mqtt.Subscribe(fmt.Sprintf("vclock/register/%s", simulation.name), recvRegister)

	// インスタンス登録のSubscribe
	for _, simulator := range simulation.simulators {
		mqtt.Subscribe(fmt.Sprintf("vclock/instantiate/%s/%s", simulation.name, simulator.name), recvInstantiate)
	}

	// 実行準備完了（Ready）のSubscribe
	for _, simulator := range simulation.simulators {
		mqtt.Subscribe(fmt.Sprintf("vclock/ready/%s/%s", simulation.name, simulator.name), recvReady)
	}

	// 実行完了（Done）のSubscribe
	for _, simulator := range simulation.simulators {
		mqtt.Subscribe(fmt.Sprintf("vclock/done/%s/%s", simulation.name, simulator.name), recvDone)
	}

	// 実行終了（Complete）のSubscribe
	for _, simulator := range simulation.simulators {
		mqtt.Subscribe(fmt.Sprintf("vclock/complete/%s/%s", simulation.name, simulator.name), recvComplete)
	}

	///////// シミュレーション準備サイクル /////////
	// すべてのシミュレータのインスタンスが起動したかどうか確認
	for {
		cwg.Add(1)
		cwg.Wait()
		if simulation.ReadyToRun() {
			fmt.Println("vcserver received instantiate requests from all instances")
			break
		}
	}

	///////// シミュレーション実行サイクル /////////
	for {
		fmt.Printf("Simulation clock %d ready --------- \n", globalclock)
		logEvent("GlobalClock", "VirtualClock", -1, globalclock, "Ready")

		// 実行準備完了を確認
		for {
			if simulation.isReadyAll(globalclock) {
				break
			} else {
				fmt.Println("... waiting for ready")
				cwg.Add(1)
				cwg.Wait()
			}
		}

		fmt.Printf("Simulation clock %d start --------- \n", globalclock)
		logEvent("GlobalClock", "VirtualClock", -1, globalclock, "Start")

		// １シミュレーションサイクルを実行
		for _, simulator := range simulation.simulators {
			bytes, _ := json.Marshal(VClockMessageTypes.State{
				Id:    -1,
				Clock: globalclock,
			})
			mqtt.Publish(fmt.Sprintf("vclock/run/%s/%s", simulation.name, simulator.name), bytes)
		}

		// 全インスタンスの実行完了を確認
		// (シミュレーションの実行時刻でないインスタンスからは Done を受け取らない)
		for {
			if simulation.isDoneAll(globalclock) {
				for _, simulator := range simulation.simulators {
					//「次のステップ」メッセージを送信
					bytes, _ := json.Marshal(VClockMessageTypes.State{
						Id:    -1,
						Clock: globalclock,
					})
					mqtt.Publish(fmt.Sprintf("vclock/next/%s/%s", simulation.name, simulator.name), bytes)
				}
				break
			} else {
				cwg.Add(1)
				cwg.Wait()
			}
		}
		fmt.Printf("Simulation clock %d done --------- \n", globalclock)
		logEvent("GlobalClock", "VirtualClock", -1, globalclock, "Done")

		// 次のシミュレーション実行時刻を設定
		simulation.updateNextClock(globalclock)
		// 次のシミュレーションサイクルに移行
		globalclock++
	}
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
