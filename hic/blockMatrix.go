package hic

import (
	"bytes"
	"fmt"
	"log"
	"math"
	//"sort"
	"sync"
	"time"
	"unsafe"

	"github.com/gonum/matrix/mat64"
	. "github.com/nimezhu/netio"
)

const (
	MaxCells           = 25000000 //5000*5000
	maxBlockBufferSize = 1
)

/* BlockMatrix : implement mat64.Matrix interface
 */
type BlockMatrix struct {
	Unit              string
	ResIdx            int32
	SumCounts         float32
	occupiedCellCount float32 //not use now
	stdDev            float32 //not use now
	percent95         float32 //not use now
	BinSize           int32
	BlockBinCount     int32
	BlockColumnCount  int32
	BlockCount        int32
	BlockIndexes      map[int]BlockIndex
	buffers           map[int]*Block
	lastUsedDate      map[int]time.Time
	bufferMux         sync.Mutex
	mux               sync.Mutex
	reader            MutexReadSeeker //TODO change to io.ReadSeeker and Postion Index
	r                 int
	c                 int
	useBuffer         bool
}

func (b *BlockMatrix) Matrix() mat64.Matrix {
	return b.View(0, 0, b.r, b.c)
}
func (b *BlockMatrix) Dims() (int, int) {
	return b.r, b.c
}
func (b *BlockMatrix) String() string {
	var s bytes.Buffer
	s.WriteString("BlockMatrix\n")
	s.WriteString("\tunit　\t")
	s.WriteString(b.Unit)
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("\tresIdx\t%d\n", b.ResIdx))
	s.WriteString(fmt.Sprintf("\tsumCounts\t%.2f\n", b.SumCounts))
	s.WriteString(fmt.Sprintf("\tbinSize\t%d\n", b.BinSize))
	s.WriteString(fmt.Sprintf("\tblockBinCount\t%d\n", b.BlockBinCount))
	s.WriteString(fmt.Sprintf("\tblockColumnCount\t%d\n", b.BlockColumnCount))
	s.WriteString(fmt.Sprintf("\tblockCount\t%d\n", b.BlockCount))
	return s.String()
}

func (b *BlockMatrix) resetBuffer(i int) {
	b.bufferMux.Lock()
	defer b.bufferMux.Unlock()
	log.Println("reset block matrix buffers")
	for k := range b.buffers {
		if k != i {
			delete(b.buffers, k)
		}
	}
	log.Println("done")
}

func (b *BlockMatrix) loadBlock(index int) bool {
	v, ok := b.BlockIndexes[index]
	if !ok {
		return ok //not correct index
	}
	signal := true
	if !b.isBlockLoaded(index) {
		b.mux.Lock()
		c := getBlock(b.reader, v.Position, v.Size)
		if c.NPositions >= 0 {
			b.buffers[index] = c
		} else {
			signal = false
		}
		b.mux.Unlock()
	}
	return signal
}

func (b *BlockMatrix) coordToBlockIndex(i int, j int) int {
	blockBinCount := int(b.BlockBinCount)
	if blockBinCount == 0 {
		return -1
	}
	row := i / blockBinCount
	col := j / blockBinCount
	return row*int(b.BlockColumnCount) + col
}

func (b *BlockMatrix) coordsToBlockIndexes(i int, j int, r int, c int) []int {
	blockBinCount := int(b.BlockBinCount)
	if blockBinCount == 0 {
		a := make([]int, 0, 0)
		return a
	}
	startrow := i / blockBinCount
	startcol := j / blockBinCount
	endrow := (i + r) / blockBinCount
	endcol := (j + c) / blockBinCount
	//log.Println("startrow,endrow", startrow, endrow)
	//log.Println("startcol,endcol", startcol, endcol)
	if endcol > int(b.BlockColumnCount) {
		endcol = int(b.BlockColumnCount)
	} //TODO Fix endrow > blocks
	//fmt.Println(startrow, endrow, startcol, endcol, b.BlockColumnCount)
	arr := make([]int, 0, (endrow-startrow+1)*(endcol-startcol+1))
	for i := startrow; i <= endrow; i++ {
		for j := startcol; j <= endcol; j++ {
			idx := i*int(b.BlockColumnCount) + j //TODO DEBUG??? sort chromosome ???
			_, ok := b.BlockIndexes[idx]
			if !ok {
				idx = max(i, j)*int(b.BlockColumnCount) + min(i, j) //if symetric
			}
			_, ok = b.BlockIndexes[idx]
			if ok {
				arr = append(arr, idx)
			} else {
				log.Println("not ok loading", idx) //TODO Handler
			}
		}
	}
	return arr
}

func (b *BlockMatrix) isBlockLoaded(i int) bool {
	_, ok := b.buffers[i]
	return ok
}
func (b *BlockMatrix) loadBlocks(indexes []int) {
	for _, v := range indexes {
		b.loadBlock(v)
	}
}
func (b *BlockMatrix) At(i int, j int) float64 {
	block := b.coordToBlockIndex(i, j)
	if block == -1 {
		return math.NaN()
	}
	if !b.useBuffer && len(b.buffers) > 5 { //mininum TO be test
		go func() {
			log.Println("reset buffer")
			b.resetBuffer(block)
		}()
	}
	_, ok := b.BlockIndexes[block]
	if ok {
		b.loadBlock(block)
		x := i - int(b.buffers[block].XOffset)
		y := j - int(b.buffers[block].YOffset)
		r, c := b.buffers[block].Dims()
		if x >= 0 && x < r && y >= 0 && y < c {
			return b.buffers[block].At(x, y)
		}
	}
	//log.Println("Not ok") //TODO RM
	return math.NaN() //TODO
}
func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

type timeIndex struct {
	key  int
	date time.Time
}
type timeSlice []timeIndex

func (p timeSlice) Len() int {
	return len(p)
}

func (p timeSlice) Less(i, j int) bool {
	return p[i].date.Before(p[j].date)
}

func (p timeSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

/* clean buffer
 *	 clear 1 hour used Before
 * 	 if still > 2/3 max size
 *    clear 1/3 max sorted buffer
 *  TO Test
 */
func (b *BlockMatrix) cleanBuffer() {
	//var dk = []int{}
	b.bufferMux.Lock()
	defer b.bufferMux.Unlock()
	/*
		for k, v := range b.lastUsedDate {
			diff := time.Now().Sub(v)
			if diff.Hours() > 1.0 {
				delete(b.buffers, k) //TODO Test it in the field
				dk = append(dk, k)
			}
		}
	*/
	//log.Println("buffers length", len(b.buffers))
	//log.Println("maxBlockBufferSize", maxBlockBufferSize)
	for k, v := range b.buffers {
		v.FreeMem()
		delete(b.buffers, k)
		delete(b.lastUsedDate, k)
		//log.Println("delete", k)
		v = nil
	}
	/*
		if len(b.buffers) > maxBlockBufferSize*2/3 {
			timeIndexes := make(timeSlice, 0, len(b.buffers))
			for k, _ := range b.buffers {
				date, _ := b.lastUsedDate[k]
				timeIndexes = append(timeIndexes, timeIndex{k, date})
				log.Println(k)
			}
			sort.Sort(timeIndexes)
			for i := 0; i < maxBlockBufferSize/3; i++ {
				delete(b.buffers, timeIndexes[i].key)
				log.Println("delete", timeIndexes[i].key)
				dk = append(dk, timeIndexes[i].key)
			}
		}
		for _, v := range dk {
			delete(b.lastUsedDate, v)
		}
	*/
	//Add Buffer Mutex to Clean

}
func (b *BlockMatrix) SetUseBuffer(a bool) {
	b.useBuffer = a
}
func (b *BlockMatrix) View(i int, j int, r int, c int) mat64.Matrix {
	if r*c > MaxCells {
		return nil
	}
	if b.useBuffer && len(b.buffers) > maxBlockBufferSize {
		go func() {
			//log.Println("clean buffer")
			b.cleanBuffer() //TODO with Syntax.
		}()
	}
	mat := mat64.NewDense(r, c, make([]float64, r*c))
	blocks := b.coordsToBlockIndexes(i, j, r, c)
	//log.Println(blocks)
	if b.useBuffer {
		//log.Println("using buffer")
		b.bufferMux.Lock() //TODO ADD Buffer Lock Instead
		b.loadBlocks(blocks)
		for _, index := range blocks {
			v, ok := b.buffers[index] //TODO fix write same time
			if !ok {

			}
			b.lastUsedDate[index] = time.Now()
			vr, vc := v.Dims()
			xoffset := int(v.XOffset)
			yoffset := int(v.YOffset)
			for x := max(xoffset, i); x < min(xoffset+vr, i+r); x++ {
				for y := max(yoffset, j); y < min(yoffset+vc, j+c); y++ {
					if math.Abs(mat.At(x-i, y-j)) < math.Abs(v.At(x-xoffset, y-yoffset)) { //TODO override
						mat.Set(x-i, y-j, v.At(x-xoffset, y-yoffset))
					}
				}
			}
		}
		b.bufferMux.Unlock()
	} else {
		b.mux.Lock()
		//log.Println("not using buffer")
		for _, index := range blocks {
			v0, ok := b.BlockIndexes[index]
			if !ok {
				return nil
			}
			v := getBlock(b.reader, v0.Position, v0.Size)
			vr, vc := v.Dims()
			xoffset := int(v.XOffset)
			yoffset := int(v.YOffset)
			for x := max(xoffset, i); x < min(xoffset+vr, i+r); x++ {
				for y := max(yoffset, j); y < min(yoffset+vc, j+c); y++ {
					if math.Abs(mat.At(x-i, y-j)) < math.Abs(v.At(x-xoffset, y-yoffset)) { //TODO override
						mat.Set(x-i, y-j, v.At(x-xoffset, y-yoffset))
					}
				}
			}
		}
		b.mux.Unlock()
	}

	return mat
}

/* BufferSize */

func (b *BlockMatrix) BufferSize() uintptr {
	return unsafe.Sizeof(b.buffers)
}
