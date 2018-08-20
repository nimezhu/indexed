package main

import (
	"fmt"
	"os"

	. "github.com/nimezhu/indexed/bbi"
	"github.com/nimezhu/netio"

	ui "github.com/gizak/termui"
) // <- ui shortcut, optionaal
func regionText(chr string, start int, end int) string {
	return fmt.Sprintf("%s:%d-%d", chr, start, end)
}
func main() {
	err := ui.Init()
	if err != nil {
		panic(err)
	}
	defer ui.Close()

	p := ui.NewPar("chr1:1000000-2000000")
	p.Height = 3
	p.Width = 50
	p.TextFgColor = ui.ColorWhite
	p.BorderLabel = "Text Box"
	p.BorderFg = ui.ColorCyan

	maxDiv := ui.NewPar("")
	maxDiv.Height = 3
	maxDiv.BorderLabel = "max"
	maxDiv.Width = 20
	maxDiv.Y = 5
	minDiv := ui.NewPar("")
	minDiv.BorderLabel = "min"
	minDiv.Height = 3
	minDiv.Width = 20
	minDiv.Y = 8

	bw := initBw(os.Args[1])

	spl := ui.NewSparkline()

	/* update spl */
	update := func(chr string, start int, end int) {
		v := queryBw(bw, chr, start, end)
		data := make([]int, len(v))
		p.Text = regionText(chr, start, end)
		max := v[0]
		min := v[0]
		for i, v0 := range v {
			data[i] = int(v0 * 100)
			if max < v0 {
				max = v0
			}
			if min > v0 {
				min = v0
			}
		}
		spl.Data = data
		spl.Title = os.Args[1]
		spl.LineColor = ui.ColorGreen
		spls := ui.NewSparklines(spl) //...
		spls.Height = 5
		spls.Width = 100
		spls.Y = 5
		spls.X = 20

		maxDiv.Text = fmt.Sprintf("%.2f", max)
		minDiv.Text = fmt.Sprintf("%.2f", min)
		ui.Render(p, spls, maxDiv, minDiv) // feel free to call Render, it's async and non-blocks
	} // event handler...
	start := 1000000
	end := 2000000
	update("chr1", start, end)
	ui.Handle("/sys/kbd/q", func(ui.Event) {
		// press q to quit
		ui.StopLoop()
	})
	ui.Handle("/sys/kbd/j", func(ui.Event) {
		l := end - start
		start -= l / 2
		end -= l / 2
		update("chr1", start, end)
	})
	ui.Handle("sys/kbd/l", func(ui.Event) {
		l := end - start
		start += l / 2
		end += l / 2
		update("chr1", start, end)

	})
	ui.Handle("/sys/kbd/g", func(ui.Event) {

	})
	ui.Loop()
}
func initBw(fn string) *BigWigReader {
	f, _ := netio.NewReadSeeker(fn)
	//f.BufferSize(8192)
	bwf := NewBbiReader(f)
	bwf.InitIndex()
	bw := NewBigWigReader(bwf)
	return bw
}
func queryBw(bw *BigWigReader, chr string, start int, end int) []float64 {
	a, err := bw.Query(chr, start, end, 50)
	if err != nil {
		panic(err)
	}
	retv := make([]float64, 0, 60)
	for v := range a {

		retv = append(retv, v.Sum/float64(v.Valid))
	}
	return retv
}
