package hic

import (
	//"fmt"
	//"errors"
	"errors"
	"fmt"

	"github.com/gonum/matrix/mat64"
	//"log"
	"strconv"
)

var units = []string{"BP", "FRAG"}

func (e *HiC) queryVector(a bed3, normtype int, unit int, resIdx int) ([]float64, error) {
	binSize := e.BpRes[resIdx]
	start := a.start / int(binSize)
	end := (a.end-1)/int(binSize) + 1
	//key := strconv.Itoa(e.chr2idx(a.chr)) + "_" + strconv.Itoa(e.chr2idx(a.chr))
	idx := e.chr2idx(a.chr)
	if idx == -1 {
		return nil, errors.New("chromosome not found")
	}
	key := strconv.Itoa(normtype) + "_" + strconv.Itoa(e.chr2idx(a.chr)) + "_" + strconv.Itoa(unit) + "_" + strconv.Itoa(int(binSize))
	v, ok := e.Footer.NormVector[key]

	if ok {
		v.Load(e)
		values, _ := v.Data()
		//fmt.Println("vector length", len(values))
		//fmt.Println("vector", values)

		return values[start:end], nil
	}

	return nil, errors.New("value not found")
}

func (e *HiC) QueryOneNormMat(chr string, start int, end int, resIdx int, normtype int, unit int) (mat64.Matrix, error) {
	return e.queryOneNormMat(bed3{chr, start, end}, resIdx, normtype, unit)
}
func (e *HiC) queryOneNormMat(a bed3, resIdx int, normtype int, unit int) (mat64.Matrix, error) { //TODO check if the matrix is sparse
	m, err := e._queryOne(a, resIdx)
	if err != nil {
		fmt.Println("_queryOne")
		return nil, err
	}
	vec, err := e.queryVector(a, normtype, unit, resIdx)
	if err != nil {
		fmt.Println(unit, normtype, "_notFoundVector")
		return nil, err
	}
	r, c := m.Dims()
	if len(vec) > 0 {
		retv := mat64.NewDense(r, c, make([]float64, r*c))
		for i := 0; i < r; i++ {
			for j := 0; j < c; j++ {
				retv.Set(i, j, m.At(i, j)/(vec[i]*vec[j]))
			}
		}
		return retv, nil
	} else {
		return m, nil
	}
}

func (e *HiC) QueryTwoNormMat(chr string, start int, end int, chr2 string, start2 int, end2 int, resIdx int, normtype int, unit int) (mat64.Matrix, error) {
	return e.queryTwoNormMat(bed3{chr, start, end}, bed3{chr2, start2, end2}, resIdx, normtype, unit)
}
func (e *HiC) queryTwoNormMat(a bed3, b bed3, resIdx int, normtype int, unit int) (mat64.Matrix, error) { //TODO check if the matrix is sparse
	m, err := e._queryTwo(a, b, resIdx)
	if err != nil {
		return nil, err
	}
	vecA, err := e.queryVector(a, normtype, unit, resIdx)
	if err != nil {
		return nil, err
	}
	vecB, err := e.queryVector(b, normtype, unit, resIdx)
	if err != nil {
		return nil, err
	}
	r, c := m.Dims()
	if len(vecA) > 0 && len(vecB) > 0 {
		retv := mat64.NewDense(r, c, make([]float64, r*c))
		for i := 0; i < r; i++ {
			for j := 0; j < c; j++ {
				if vecA[i] > 0 && vecB[j] > 0 {
					retv.Set(i, j, m.At(i, j)/(vecA[i]*vecB[j]))
				} else {
					retv.Set(i, j, m.At(i, j)/((vecA[i]+1.0)*(vecB[j]+1.0))) //TODO psuedo count
				}
			}
		}
		return retv, nil
	} else {
		return m, nil //return norm 0, no vector found.
	}
}

func (e *HiC) queryOneFoldChangeOverExpected(a bed3, normtype int, unit int, resIdx int) (mat64.Matrix, error) {
	m, err := e.queryOneNormMat(a, resIdx, normtype, unit)
	if err != nil {
		fmt.Println(a, "error in query one norm mat", err)
		return nil, err
	}
	r, c := m.Dims()
	chrIdx := e.chr2idx(a.chr)
	binSize := e.BpRes[resIdx]
	ekey := units[unit] + "_" + strconv.Itoa(int(binSize)) + "_" + strconv.Itoa(normtype)
	//fmt.Println("ekey", ekey)
	expt, ok := e.Footer.ExpectedValueMap[ekey]
	if ok {
		newM := mat64.NewDense(r, c, make([]float64, r*c))
		for i := 0; i < r; i++ {
			for j := i; j < c; j++ {
				newM.Set(i, j, m.At(i, j)/expt.ExpectedValue(int32(chrIdx), j-i))
			}
		}

		return newM, nil
	} else {
		//fmt.Println(e.Footer.ExpectedValueMap)
		panic("not found")
	}
	return nil, errors.New("oe")
}
func (e *HiC) QueryOE(chr string, start int, end int, normtype int, unit int, resIdx int) (mat64.Matrix, error) {
	fmt.Println("in query oe")
	return e.queryOneFoldChangeOverExpected(bed3{chr, start, end}, normtype, unit, resIdx)
}

func (e *HiC) queryTwoFoldChangeOverExpected(a bed3, b bed3, normtype int, unit int, resIdx int) (mat64.Matrix, error) {
	if a.chr != b.chr {
		return nil, errors.New("not in the same chromosome")
	}
	m, err := e.queryTwoNormMat(a, b, resIdx, normtype, unit)
	if err != nil {
		fmt.Println(a, "error in query one norm mat", err)
		return nil, err
	}
	r, c := m.Dims()
	chrIdx := e.chr2idx(a.chr)
	binSize := e.BpRes[resIdx]
	ekey := units[unit] + "_" + strconv.Itoa(int(binSize)) + "_" + strconv.Itoa(normtype)
	//fmt.Println("ekey", ekey)
	expt, ok := e.Footer.ExpectedValueMap[ekey]
	x, _ := e.Corrected(a.start, a.end, resIdx)
	y, _ := e.Corrected(b.start, b.end, resIdx)
	x = x / int(e.BpRes[resIdx])
	y = y / int(e.BpRes[resIdx])
	if ok {
		newM := mat64.NewDense(r, c, make([]float64, r*c))
		for i := 0; i < r; i++ {
			for j := 0; j < c; j++ {
				d := x + i - (y + j)
				if d < 0 {
					d = -d
				}
				newM.Set(i, j, m.At(i, j)/expt.ExpectedValue(int32(chrIdx), d))
			}
		}

		return newM, nil
	} else {
		//fmt.Println(e.Footer.ExpectedValueMap)
		panic("not found")
	}
	return nil, errors.New("oe")
}

func (e *HiC) QueryOE2(chr string, start int, end int, chr2 string, start2 int, end2 int, normtype int, unit int, resIdx int) (mat64.Matrix, error) {
	return e.queryTwoFoldChangeOverExpected(bed3{chr, start, end}, bed3{chr2, start2, end2}, normtype, unit, resIdx)
}
