package main

import (
	"VClock"
	"fmt"
)

// 避難者の状態
type Status int

const (
	Stay Status = iota // 移動前
	Move               // 移動中
	Dead               // 被災
	Exit               // 避難完了
)

type Point struct{ x, y int }

// 避難者情報
type Person struct {
	id     int         // 個人ID
	curr   Point       // 現在地
	dest   Point       // 避難所
	status Status      // 避難者の状態
	ch     chan Notify // 状態報告のための通信路
}

// 報告する情報
type Notify struct {
	id     int    // 個人ID
	curr   Point  // 現在地
	status Status // 避難者の状態
}

func printStatus(notify Notify) {
	fmt.Printf("\tID : %d, Location: (%d, %d) ", notify.id, notify.curr.x, notify.curr.y)
	switch notify.status {
	case Stay:
		fmt.Printf("Status: Stay \n")
	case Move:
		fmt.Printf("Status: Move \n")
	case Dead:
		fmt.Printf("Status: Dead \n")
	case Exit:
		fmt.Printf("Status: Exit \n")
	default:
		fmt.Printf("Status: Unknown \n")
	}
}

// 避難者の動作
func (p *Person) action(id int, vc *VClock.VClock) {
	for {
		// ステップ実行開始
		vc.StepBegin(id)

		// 移動状態更新
		p.status = Move
		p.curr.x += 1
		p.curr.y += 1

		// 避難所到着確認
		if p.curr == p.dest {
			p.status = Exit
		}

		// 状態を通知
		p.ch <- Notify{p.id, p.curr, p.status}

		// ステップ実行終了
		vc.StepEnd(id)

		// 避難所に到着したら、サブルーチン終了
		if p.status == Exit {

			// ステップ実行の完了通知
			vc.Complete(id)
			break
		}
	}
}

func main() {
	// 避難者の人数
	evacuees := 3

	// 状態通知の通信路
	ch := make(chan Notify)

	// 避難者ステータスの初期化
	person := make([]Person, evacuees)
	for i := range person {
		person[i].id = i
		person[i].curr = Point{0, 0}
		person[i].dest = Point{10, 10}
		person[i].status = Stay
		person[i].ch = ch
	}

	vc := VClock.Initialize("../../config/mqtt.conf")
	// シミュレータで利用するスレッド数の登録
	idlist, err := vc.Register(evacuees)
	if err != nil {
		fmt.Print(err.Error())
	}

	// 避難実行
	for i := range person {
		go person[i].action(idlist[i], vc)
	}

	for evacuees > 0 {
		// 各避難者の状態を受信
		s := <-ch
		if s.status == Exit {
			evacuees--
		} else {
			// 各避難者の状況を表示
			printStatus(s)
		}
	}
	//　シミュレーションの完了を通知
	// （go ルーチンよりも親スレッドが先に終わる場合は、CompleteAll で全スレッドの終了を通知
	vc.CompleteAll()
}
