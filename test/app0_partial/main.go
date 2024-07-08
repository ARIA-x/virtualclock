package main

import (
	"VClock"
	"fmt"
)

type MotionSim struct {
	t float64
	h float64
}

func main() {

	var results []MotionSim

	_g := 9.8   // gravity
	_v0 := 10.0 // initial velocity
	_h0 := 0.0  // initial height
	_dt := 0.1  // clock interval

	_t := 0.0 // start time
	_h := _h0 // start height

	vc := VClock.Initialize("../../config/mqtt.conf")
	// シミュレータで利用するスレッド数の登録
	idList, err := vc.Register(1)
	if err != nil {
		fmt.Print(err.Error())
	}

	// discretized expressions
	for _h >= _h0 {
		// 周期タスクの実行開始
		vc.StepBegin(idList[0])
		results = append(results, MotionSim{_t, _h})

		_h += (-_g*_t + _v0) * _dt
		_t += _dt
		// 周期タスクの実行終了
		vc.StepEnd(idList[0])
	}
	// シミュレータの実行完了
	vc.Complete(idList[0])
	fmt.Print(results)
}
