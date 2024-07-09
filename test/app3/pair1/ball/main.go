package main

import (
	"VClock"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
)

type Position struct {
	T float64 `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type MotionSim struct {
	t float64
	x float64
	y float64
	z float64
}

type HorizontalTrajectory struct {
	angle float64
}

func netInit(socket string) net.Conn {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func (p *HorizontalTrajectory) position(x float64) (float64, float64) {
	radian := p.angle * math.Pi / 180
	y := 1.0 / 3.0 * math.Sin(x)
	xd := math.Cos(radian)*x - math.Sin(radian)*y
	yd := math.Sin(radian)*x + math.Cos(radian)*y
	return xd, yd
}

func send(t float64, x float64, y float64, z float64) {
	// conn := netInit("C:\\Users\\kohig\\AppData\\Local\\Temp\\catch_the_fly3")
	conn := netInit("/aria-dsl2/catch_the_fly1")
	// send data
	bytes, _ := json.Marshal(Position{
		T: t,
		X: x,
		Y: y,
		Z: z,
	})
	conn.Write(bytes)
	conn.Close()
}

type Params struct {
	g       float64
	v0      float64
	h0      float64
	dt      float64
	t       float64
	h       float64
	x       float64
	xw      float64
	yw      float64
	ht      HorizontalTrajectory
	results []MotionSim
}

func (p *Params) motion() {
	// discretized expressions
	p.results = append(p.results, MotionSim{p.t, p.xw, p.yw, p.h})
	send(p.t, p.xw, p.yw, p.h)
	// vertical trajectory
	p.h += (-p.g*p.t + p.v0) * p.dt
	p.t += p.dt

	// horizontal trajectory
	// (adding pseudo wind effect)
	p.x += p.v0 * p.dt
	p.xw, p.yw = p.ht.position(p.x)
}

func (p *Params) condition() bool {
	if p.h < p.h0 {
		fmt.Println(p.h, p.h0)
		return true
	} else {
		return false
	}
}

func main() {

	var p Params

	p.ht = HorizontalTrajectory{45}

	p.g = 9.8   // gravity
	p.v0 = 10.0 // initial velocity
	p.h0 = 0.0  // initial height
	p.dt = 0.1  // clock interval

	p.t = 0.0  // start time
	p.h = p.h0 // start height

	p.x = 0.0 // x position (linear) y=x

	p.xw = 0.0 // x position (with pseudo wind effect)
	p.yw = 0.0 // y position (with pseudo wind effect)

	vc := VClock.Initialize("../../../../config/mqtt.conf")
	// シミュレータで利用するスレッド数の登録
	idList, err := vc.Register(1)
	if err != nil {
		fmt.Print(err.Error())
	}

	// シミュレーション時間に ClockCycle をかけておく
	p.dt = 0.1 * float64(vc.ClockCycle)

	// シミュレーションタスクの委任
	// (シミュレーション関数, 終了条件)
	vc.Delegate(idList[0], p.motion, p.condition)

	// シミュレーションの最終結果を送信
	//（粒度が粗いとボールが空中で止まった状態になるのを防ぐため）
	// TODO: シミュレーションの終了シーケンスを改善
	send(p.t, p.xw, p.yw, p.h)

	// シミュレーション結果の確認
	fmt.Printf("Simulation has done, ")

	fmt.Print(p.results)
	fmt.Scanln()

	//drawResult(results)
}
