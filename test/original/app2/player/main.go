package main

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

// 現在のボール位置
var ballPosition = Position{-1, -1, -1, -1}

// 落下予測地点
var estimatePoint = Point{-1, -1}

// 落下予想時刻
var estimateTime = -1.0

// ステップ時間
var deltaT = 0.01

var wg sync.WaitGroup

// 選手の状態
type Status int

const (
	Stay    Status = iota // 移動前
	Move                  // 移動中
	Fail                  // キャッチ失敗
	Success               // キャッチ成功
)

type Position struct {
	T float64 `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Point struct{ x, y float64 }

// 選手情報
type Person struct {
	id        int     // 個人ID
	curr      Point   // 現在地
	status    Status  // 選手の状態
	maxSpeedX float64 // 選手の最大スピード(X方向)
	maxSpeedY float64 // 選手の最大スピード(Y方向)
	curSpeedX float64 // 選手の現在スピード(X方向)
	curSpeedY float64 // 選手の現在スピード(Y方向)
	clock     float64 // ローカルクロック
	trace     []Point // 移動軌跡

}

// 報告する情報
type Notify struct {
	id     int    // 個人ID
	curr   Point  // 現在地
	status Status // 選手の状態
}

// Point 構造体の演算子
func (p1 *Point) eq(p2 Point) bool {
	return (p1.x == p2.x && p1.y == p2.y)
}

func (p1 *Point) gt(p2 Point) bool {
	return (p1.x > p2.x && p1.y > p2.y)
}

func (p1 *Point) ge(p2 Point) bool {
	return (p1.x >= p2.x && p1.y >= p2.y)
}

func (p1 *Point) le(p2 Point) bool {
	return (p1.x <= p2.x && p1.y <= p2.y)
}

func (p1 *Point) lt(p2 Point) bool {
	return (p1.x < p2.x && p1.y < p2.y)
}

func (p1 *Point) ne(p2 Point) bool {
	return (p1.x != p2.x && p1.y != p2.y)
}

// 選手の動作
func (p *Person) action(c chan Notify) {
	for {
		if estimateTime == -1 {
			//TODO: remove busy loop
			continue
		}

		d := distance(estimatePoint.x-p.curr.x, estimatePoint.y-p.curr.y)

		if d <= 0.3 {
			p.status = Stay
		} else {
			// 移動状態更新
			p.status = Move
			p.curr.x += p.curSpeedX * deltaT
			p.curr.y += p.curSpeedY * deltaT
		}

		if d > 30.0 {
			break
		}

		// ボールがキャッチできる範囲にあるかどうか確認
		// （選手のグローブは Z軸 1.0 の高さにある想定）
		d3 := distance3(ballPosition.X-p.curr.x, ballPosition.Y-p.curr.y, ballPosition.Z-1.0)
		print("Distance :", d3, "\n")
		if d3 <= 0.2 {
			fmt.Print("Success !!!!!!!!!!!!!\n")
			p.status = Success
		} else if estimateTime <= 0.1 && ballPosition.Z <= 0.5 {
			fmt.Print("Fail !!!!!!!!!!!!!\n")
			p.status = Fail
		}

		// 予測落下時刻と予想落下地点から移動スピードを更新
		p.curSpeedX = math.Min(p.maxSpeedX, (estimatePoint.x-p.curr.x)/estimateTime)
		p.curSpeedY = math.Min(p.maxSpeedY, (estimatePoint.y-p.curr.y)/estimateTime)

		fmt.Print("SPEEDDDDD ", p.curSpeedX, p.curSpeedY, "\n")

		// 状態を通知
		c <- Notify{p.id, p.curr, p.status}

		// ボールのキャッチに成功・失敗したら、サブルーチン終了
		if p.status == Fail || p.status == Success {
			break
		}
	}
}

// 高さ方向の移動距離と時間から落下予想時刻を計算する
func timeEstimation(prev Position, next Position) float64 {
	_g := 9.8
	_dt := next.T - prev.T
	_h0 := prev.Z
	_v0 := (next.Z - prev.Z) / _dt
	_t := 0.0
	_h := _h0

	// 投げ上げシミュレーション
	for _h >= _h0 {
		// vertical trajectory
		_h += (-_g*_t + _v0) * _dt
		_t += _dt
	}
	return _t
}

func distance(x float64, y float64) float64 {
	return math.Sqrt(math.Pow(x, 2) + math.Pow(y, 2))
}

func distance3(x float64, y float64, z float64) float64 {
	return math.Sqrt(math.Pow(x, 2) + math.Pow(y, 2) + math.Pow(z, 2))
}

// 水平方向の移動距離と時間から落下予想地点を計算する
func pointEstimation(prev Position, next Position, estimateTime float64) Point {
	dx := next.X - prev.X
	dy := next.Y - prev.Y
	dt := next.T - prev.T

	vx := dx / dt
	vy := dy / dt

	return Point{vx*estimateTime + next.X, vy*estimateTime + next.Y}
}

func update(conn net.Conn) {
	defer conn.Close()
	//fmt.Printf("Connected: %s\n", conn.RemoteAddr().Network())
	buf := make([]byte, 1024)
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Print(err.Error())
			}
			return
		}
		data := buf[0:nr]
		var pos Position
		json.Unmarshal(data, &pos)
		fmt.Printf("Received :T %f, X %f, Y %f, Z %f\n", pos.T, pos.X, pos.Y, pos.Z)

		if ballPosition.T != -1 {
			// ボールの落下予想時刻と場所を推定
			estimateTime = timeEstimation(ballPosition, pos)
			fmt.Printf("Estimate Time: %f\n", estimateTime)
			// ボールの落下予想時刻と場所を更新
			estimatePoint = pointEstimation(ballPosition, pos, estimateTime)
			fmt.Printf("Estimate Pos: %f, %f\n", estimatePoint.x, estimatePoint.y)
		}

		//ボールの位置更新
		ballPosition.T = pos.T
		ballPosition.X = pos.X
		ballPosition.Y = pos.Y
		ballPosition.Z = pos.Z
	}
}

// ボールの現在位置を受信するサーバ
func server(listener net.Listener) {
	defer listener.Close()

	fmt.Println("server launched...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Print(err.Error())
		} else {
			update(conn)
		}
	}
}

func main() {

	// Unix ソケットファイル
	socket := "/virtualclock/catch_the_fly"

	// listen ソケット準備
	close := make(chan int)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		fmt.Print(err.Error(), "\n")
	}
	// プログラム終了時の後片付け
	shutdown(listener, socket, close)

	//受信サーバの用意
	wg.Add(1)
	go server(listener)

	// 選手の人数
	evacuee := 1

	// 選手ステータスの初期化
	person := make([]Person, evacuee)
	for i := range person {
		person[i].id = i
		person[i].curr = Point{6.0, 0.0}
		person[i].status = Stay
		person[i].clock = 0.0
		person[i].curSpeedX = 0.0
		person[i].curSpeedY = 0.0
		person[i].maxSpeedX = 11.0 //maxSpeed should be bigger than ball speed.
		person[i].maxSpeedY = 11.0 //maxSpeed should be bigger than ball speed.
	}

	// 状態通知の通信路
	ch := make(chan Notify)

	// シミュレーション実行
	for i := range person {
		go person[i].action(ch)
	}

	for evacuee > 0 {
		// 各選手の状態を受信
		s := <-ch
		if s.status == Success || s.status == Fail {
			evacuee--
		} else {
			// 各選手の状況を表示
			person[s.id].trace = append(person[s.id].trace, s.curr)
			fmt.Print(s, "\n")
		}
	}
	for i := range person {
		drawGraph("Player Trace", "X axis", "Y axis", "images/player.png", person[i].trace)
	}
	// 後片付け後にメイン関数終了
	_ = <-close
}

func shutdown(listener net.Listener, tempfile string, close chan int) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		interrupt := 0
		for {
			s := <-c
			switch s {
			case os.Interrupt:
				if interrupt == 0 {
					fmt.Println("Interrupt...")
					interrupt++
					continue
				}
			}
			break
		}
		/*
			if err := listener.Close(); err != nil {
				fmt.Print(err.Error(), "\n")
			}
		*/
		if err := os.Remove(tempfile); err != nil {
			fmt.Print(err.Error(), "\n")
		}
		close <- 1
	}()
}

func drawGraph(title string, xlabel string, ylabel string, filename string, data []Point) {
	// インスタンスを生成
	p := plot.New()

	// 表示項目の設定
	p.Title.Text = title
	p.X.Label.Text = xlabel
	p.Y.Label.Text = ylabel

	pts := make(plotter.XYs, len(data))

	for i, axis := range data {
		pts[i].X = axis.x
		pts[i].Y = axis.y
	}
	// グラフを描画
	err := plotutil.AddLinePoints(p, pts)
	if err != nil {
		panic(err)
	}

	// 描画結果を保存
	// "5*vg.Inch" の数値を変更すれば，保存する画像のサイズを調整できます．
	if err := p.Save(6*vg.Inch, 6*vg.Inch, filename); err != nil {
		panic(err)
	}
}
