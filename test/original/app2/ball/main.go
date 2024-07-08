package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
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

type pos struct {
	x float64
	y float64
}

func netInit(socket string) net.Conn {
	conn, err := net.Dial("unix", socket)
	if err != nil {
		log.Fatal(err)
	}
	return conn
}

func drawResult(results []MotionSim) {

	data := make([]pos, len(results))
	for _, v := range results {
		data = append(data, pos{v.x, v.z})
	}
	drawGraph("Vertical trajectory", "X axis", "Z axis", "images/vertical.png", data)

	data2 := make([]pos, len(results))
	for _, v := range results {
		data2 = append(data2, pos{v.x, v.y})
	}
	drawGraph("Horizontal trajectory", "X axis", "Y axis", "images/horizontal.png", data2)
}

func drawGraph(title string, xlabel string, ylabel string, filename string, data []pos) {
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

func (p *HorizontalTrajectory) position(x float64) (float64, float64) {
	radian := p.angle * math.Pi / 180
	y := 1.0 / 3.0 * math.Sin(x)
	xd := math.Cos(radian)*x - math.Sin(radian)*y
	yd := math.Sin(radian)*x + math.Cos(radian)*y
	return xd, yd
}

func send(t float64, x float64, y float64, z float64) {
	// conn := netInit("C:\\Users\\kohig\\AppData\\Local\\Temp\\catch_the_fly3")
	conn := netInit("/virtualclock/catch_the_fly")
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

func main() {

	var results []MotionSim

	ht := HorizontalTrajectory{45}

	_g := 9.8   // gravity
	_v0 := 10.0 // initial velocity
	_h0 := 0.0  // initial height
	_dt := 0.01 // clock interval

	_t := 0.0 // start time
	_h := _h0 // start height

	_x := 0.0 // x position (linear) y=x

	_xw := 0.0 // x position (with pseudo wind effect)
	_yw := 0.0 // y position (with pseudo wind effect)

	// discretized expressions
	for _h >= _h0 {
		results = append(results, MotionSim{_t, _xw, _yw, _h})
		send(_t, _xw, _yw, _h)
		// vertical trajectory
		_h += (-_g*_t + _v0) * _dt
		_t += _dt

		// horizontal trajectory
		// (adding pseudo wind effect)
		_x += _v0 * _dt
		_xw, _yw = ht.position(_x)
	}

	fmt.Print(results)
	drawResult(results)
}
