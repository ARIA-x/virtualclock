package main

import (
	"VClock"
	"fmt"
)

type MotionSim struct {
	t float64
	h float64
}

type Params struct {
	g       float64
	v0      float64
	h0      float64
	dt      float64
	t       float64
	h       float64
	results []MotionSim
}

func (p *Params) motion() {
	p.results = append(p.results, MotionSim{p.t, p.h})
	// discretized expressions
	p.h += (-p.g*p.t + p.v0) * p.dt
	p.t += p.dt
}

func (p *Params) condition() bool {
	if p.h < p.h0 {
		return true
	} else {
		return false
	}
}

func main() {

	var par Params

	par.g = 9.8   // gravity
	par.v0 = 10.0 // initial velocity
	par.h0 = 0.0  // initial height
	par.dt = 0.1  // clock interval

	par.t = 0.0    // start time
	par.h = par.h0 // start height

	vc := VClock.Initialize("../../config/mqtt.conf")
	// シミュレータで利用するスレッド数の登録
	idList, err := vc.Register(1)
	if err != nil {
		fmt.Print(err.Error())
	}

	// シミュレーションタスクの委任
	// (シミュレーション関数, 終了条件)
	vc.Delegate(idList[0], par.motion, par.condition)

	// シミュレーション結果の確認
	fmt.Printf("Simulation has done, ")
	fmt.Println(par.results)
	// シミュレータの実行完了
	// 入力待ち
	fmt.Scanln()
}
