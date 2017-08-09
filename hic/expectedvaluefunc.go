package hic

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/nimezhu/indexed/hic/normtype"
	"github.com/nimezhu/indexed/hic/unit"
)

type ExpectedValueFunc struct {
	normType       int //enum string
	unit           int
	binSize        int
	expectedValues []float64
	normFactors    map[int32]float64
}

func (e *ExpectedValueFunc) Key() string {
	return strconv.Itoa(e.unit) + "_" + strconv.Itoa(e.binSize) + "_" + strconv.Itoa(e.normType)
}

/*Text: output detail string
 */
func (e *ExpectedValueFunc) Text() string {
	var s bytes.Buffer
	s.WriteString(fmt.Sprintf("%s %s %d\nExprectedValues:\n", normtype.IdxToString(e.normType), unit.IdxToString(e.unit), e.binSize))
	for i := 0; i < len(e.expectedValues); i++ {
		s.WriteString(fmt.Sprintf("\t%d\t%f\n", i, e.expectedValues[i]))
	}
	s.WriteString("NormFactors:\n")
	for i, v := range e.normFactors {
		s.WriteString(fmt.Sprintf("\t%d\t%f\n", i, v))
	}
	return s.String()
}

func NewExpectedValueFunc(normType int, unit int, binSize int, expectedValues []float64, normFactors map[int32]float64) *ExpectedValueFunc {
	//normType := normtype.StringToIdx(normTypeString)
	return &ExpectedValueFunc{normType, unit, binSize, expectedValues, normFactors}
}

func (e *ExpectedValueFunc) ExpectedValues() []float64 {
	return e.expectedValues
}
func (e *ExpectedValueFunc) NormFactors() map[int32]float64 {
	return e.normFactors
}
func (e *ExpectedValueFunc) BinSize() int {
	return e.binSize
}
func (e *ExpectedValueFunc) Length() int {
	return len(e.expectedValues)
}
func (e *ExpectedValueFunc) NormTypeString() string {
	return normtype.IdxToString(e.normType)
}
func (e *ExpectedValueFunc) Unit() int {
	return e.unit
}

/*ExpectedValue: get the expected value for {chromsome, distance}
 */
func (e *ExpectedValueFunc) ExpectedValue(chrIdx int32, distance int) float64 {
	normFactor := float64(1.0)
	if e.normFactors != nil {
		if n, ok := e.normFactors[chrIdx]; ok {
			normFactor = n
		}
	}
	if distance >= len(e.expectedValues) {
		return e.expectedValues[len(e.expectedValues)-1] / normFactor
	} else {
		return e.expectedValues[distance] / normFactor
	}
}
