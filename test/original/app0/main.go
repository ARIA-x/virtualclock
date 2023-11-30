package main

import "fmt"

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

	// discretized expressions
	for _h >= _h0 {
		results = append(results, MotionSim{_t, _h})
		_h += (-_g*_t + _v0) * _dt
		_t += _dt
	}

	fmt.Print(results)
}
