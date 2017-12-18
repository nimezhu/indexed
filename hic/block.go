package hic

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"log"
	"math"

	"github.com/gonum/matrix/mat64"
	. "github.com/nimezhu/netio"
)

type MatrixViewer interface {
	mat64.Matrix
	mat64.Viewer
}
type Block struct {
	NPositions int32
	XOffset    int32
	YOffset    int32
	Mat        MatrixViewer //using dense now. todo sparse later.
}

func (b *Block) At(i int, j int) float64 {
	return b.Mat.At(i, j)
}
func (b *Block) Dims() (int, int) {
	return b.Mat.Dims()
}
func (b *Block) T() mat64.Matrix {
	return b.Mat.T()
}
func (b *Block) View(i int, j int, r int, c int) mat64.Matrix {
	return b.Mat.View(i, j, r, c)
}
func (b *Block) String() string {
	var s bytes.Buffer
	s.WriteString("Block\n")
	s.WriteString(fmt.Sprintf("\tnPositions %d\n", b.NPositions))
	s.WriteString(fmt.Sprintf("\tbinXOffSet %d\n", b.XOffset))
	s.WriteString(fmt.Sprintf("\tbinYOffSet %d\n", b.YOffset))
	r, c := b.Dims()
	s.WriteString(fmt.Sprintf("\tDims %d,%d\n", r, c))
	return s.String()
}

type Position struct {
	X     int
	Y     int
	Value float32
}

func checkErr(err error) {
	if err != nil {
		log.Println(err)
	}
}
func getBlock(e MutexReadSeeker, blockPosition int64, blockSize int32) *Block {
	chan2 := make(chan Block, 1)
	go func() {
		e.Lock()
		position, _ := e.Seek(0, 1)
		e.Seek(blockPosition, 0)
		defer e.Seek(position, 0)
		defer e.Unlock()
		b := make([]byte, blockSize)
		//l, _ := e.Read(b)
		e.Read(b)
		//fmt.Println(l, "=", len(b)) //assert format
		b0 := bytes.NewReader(b)
		c, err := zlib.NewReader(b0)
		if err != nil {
			chan2 <- Block{}
			return
		}
		nPositions, _ := ReadInt(c)
		Pos := make([]Position, nPositions) //？？？
		binXOffset, _ := ReadInt(c)
		binYOffset, _ := ReadInt(c)
		maxY := 0
		maxX := 0
		short, _ := ReadByte(c)
		useShort := (short == 0)
		//log.Println("useShort", useShort) //TODO RM
		t, _ := ReadByte(c)
		index := 0
		var X, Y int
		n := int(nPositions)
		//log.Println("nPositions", nPositions) //TODO RM
		if nPositions > 0 {
			if t == 1 { //Sparse or Triangle Sparse
				//log.Println("reading sparse") //TODO RM
				rowCount, _ := ReadShort(c)
				//log.Println("rowCount ?", rowCount) //TODO RM
				for i0 := uint16(0); i0 < rowCount; i0++ {
					//log.Println("row", i0) TODO RM
					Y16, _ := ReadShort(c)
					Y = int(Y16)
					colCount, _ := ReadShort(c)
					for j0 := uint16(0); j0 < colCount; j0++ {
						X16, _ := ReadShort(c)
						X = int(X16)
						var counts float32
						if useShort {
							a0, _ := ReadShort(c)
							counts = float32(a0)
						} else {
							a1, err := ReadFloat32(c)
							checkErr(err)
							counts = float32(a1)
							/*
								if counts < 0.000000001 {
									log.Println("warning fix", a1)
									_, err := ReadShort(c)
									checkErr(err)
									counts = float32(math.NaN())
								}
							*/

						}
						//log.Println(index)
						//log.Println(i0, j0, X, Y, counts) //TODO RM:
						/*
							if counts == 0 {
								log.Panic(i0, j0, X, Y, counts)
							}
						*/

						if index >= n {
							//log.Println(i0, j0, index, ">", nPositions) //TODO Fix Overflow for some hic data
							Pos = append(Pos, Position{X, Y, counts})

						} else {
							Pos[index] = Position{X, Y, counts}
						}
						//log.Println(X, Y, counts)
						if X > maxX {
							maxX = X
						}
						if Y > maxY {
							maxY = Y
						}

						index++
						//TODO
					}
				}
			} else if t == 2 { //Dense
				nPts, _ := ReadInt(c)
				Pos = make([]Position, nPts) //TODO test
				w0, _ := ReadShort(c)
				w := int32(w0)
				maxY = int(w0) //ColNumber
				var i0 int32
				for i0 = int32(0); i0 < nPts; i0++ {
					row := i0 / w
					col := i0 - row*w
					bin1 := binXOffset + col
					bin2 := binYOffset + row
					var counts float32
					if useShort {
						a2, _ := ReadShort(c)
						counts = float32(a2)
					} else {
						a3, _ := ReadFloat32(c)
						counts = float32(a3)
					}
					X = int(bin1)
					Y = int(bin2)
					Pos[index] = Position{X, Y, counts}
					index++
				}
				maxX = int(i0)
			}
		} else {
			log.Println("-1,-1 matrix?")
			maxX = -1
			maxY = -1
		}
		if index > n {
			log.Println("Warning", index, ">", nPositions) //TODO Fix Overflow for some hic data
		}
		m := mat64.NewDense((maxX + 1), (maxY + 1), make([]float64, (maxX+1)*(maxY+1)))
		x, y := m.Dims()
		for _, v := range Pos {
			if v.X > x-1 || v.Y > y-1 {
				//log.Println(v.X, v.Y)
				//log.Println("outbound", x, y) //TODO Fix 13000,13000 vs 2,2
			} else {
				//log.Println(i0, "set i0,x,y,v", v.X, v.Y, v.Value)         //TODO RM
				if math.Abs(m.At(v.X, v.Y)) < math.Abs(float64(v.Value)) { //TODO check Override
					m.Set(v.X, v.Y, float64(v.Value))
				}
			}
		}

		if index > n {
			//log.Println("m at 0,0 warning inside", m.At(0, 0)) //TODO RM
			chan2 <- Block{int32(index), binXOffset, binYOffset, m}
		} else {
			chan2 <- Block{nPositions, binXOffset, binYOffset, m}
		}
	}()
	b := <-chan2
	return &b
}
