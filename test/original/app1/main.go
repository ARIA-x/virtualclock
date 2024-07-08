package main

import "fmt"

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
	id     int    // 個人ID
	curr   Point  // 現在地
	dest   Point  // 避難所
	status Status // 避難者の状態
}

// 報告する情報
type Notify struct {
	id     int    // 個人ID
	curr   Point  // 現在地
	status Status // 避難者の状態
}

// 避難者の動作
func (p *Person) action(c chan Notify) {
	for {
		// 移動状態更新
		p.status = Move
		p.curr.x += 1
		p.curr.y += 1

		// 避難所到着確認
		if p.curr == p.dest {
			p.status = Exit
		}

		// 状態を通知
		c <- Notify{p.id, p.curr, p.status}

		// 避難所に到着したら、サブルーチン終了
		if p.status == Exit {
			break
		}
	}
}

func main() {

	// 避難者ステータスの初期化
	person := make([]Person, 3)
	for i := range person {
		person[i].id = i
		person[i].curr = Point{0, 0}
		person[i].dest = Point{10, 10}
		person[i].status = Stay
	}

	// 避難中の人数
	evacuee := 3

	// 状態通知の通信路
	ch := make(chan Notify)

	// 避難実行
	for i := range person {
		go person[i].action(ch)
	}

	for evacuee > 0 {
		// 各避難者の状態を受信
		s := <-ch
		if s.status == Exit {
			evacuee--
		} else {
			// 各避難者の状況を表示
			fmt.Print(s)
		}
	}
}
