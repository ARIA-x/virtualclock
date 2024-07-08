package main

import (
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

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
