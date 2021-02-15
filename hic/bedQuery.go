package hic

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gonum/matrix/mat64"
)

type bed3 struct {
	chr   string
	start int
	end   int
}

func isOverlap(a bed3, b bed3) bool {
	retv := false
	if a.chr == b.chr {
		if a.start < b.end && b.start < a.end {
			retv = true
		}
	}
	return retv
}
func merge(a bed3, b bed3) (bed3, error) {
	if !isOverlap(a, b) {
		return bed3{}, errors.New("no overlap")
	}
	return bed3{a.chr, min(a.start, b.start), max(a.end, b.end)}, nil
}
func (a bed3) length() int {
	return a.end - a.start
}

/* handle chr1 Chr1 CHR1 and 1 as same one */
func (e *HiC) chr2idx(chr string) int {
	for i, v := range e.Chr {
		a := strings.Replace(strings.ToLower(v.Name), "chr", "", -1)
		b := strings.Replace(strings.ToLower(chr), "chr", "", -1)
		if a == b {
			return i
		}
	}
	return -1 //error
}

func (e *HiC) queryTwo(a bed3, b bed3, width int) ([]mat64.Matrix, error) {
	resIdx := e.getResIdx(a.length()+b.length(), width)
	aMat, err1 := e._queryOne(a, resIdx)
	if err1 != nil {
		return nil, errors.New("chromosome not found")
	}
	bMat, err2 := e._queryOne(b, resIdx)
	if err2 != nil {
		return nil, errors.New("chromosome not found")
	}
	abMat, err3 := e._queryTwo(a, b, resIdx)
	if err3 != nil {
		return nil, errors.New("chromosome not found")
	}
	return []mat64.Matrix{aMat, bMat, abMat}, nil
}

/* QueryTwo : get two region matrix
 */
func (e *HiC) QueryTwo(chr string, start int, end int, chr2 string, start2 int, end2 int, resIdx int) (mat64.Matrix, error) { //TODO resIdx???
	if resIdx > 10 { //if resIdx > 10 , it is width of panel, change the width of panel into corresponding resIdx
		l := end - start
		if l < end2-start2 {
			l = end2 - start2
		}
		resIdx = e.getResIdx(l, resIdx)
	}
	return e._queryTwo(bed3{chr, start, end}, bed3{chr2, start2, end2}, resIdx)
}
func (e *HiC) _queryTwo(a bed3, b bed3, resIdx int) (mat64.Matrix, error) { //assert a < b
	binSize := e.BpRes[resIdx]
	aStart := a.start / int(binSize)
	aEnd := (a.end-1)/int(binSize) + 1
	aIdx := e.chr2idx(a.chr)
	bStart := b.start / int(binSize)
	bEnd := (b.end-1)/int(binSize) + 1
	bIdx := e.chr2idx(b.chr)
	if aIdx == -1 || bIdx == -1 {
		return nil, errors.New("chromosome not found")
	}
	op := false
	if aIdx > bIdx || (aIdx == bIdx && aStart > bStart) {
		aIdx, aStart, aEnd, bIdx, bStart, bEnd = bIdx, bStart, bEnd, aIdx, aStart, aEnd
		op = true
	}
	abKey := strconv.Itoa(aIdx) + "_" + strconv.Itoa(bIdx)
	abBody, err := e.loadBodyIndex(abKey)
	if err != nil {
		return nil, err
	}
	abMat := abBody.Mats[resIdx].View(aStart, bStart, aEnd-aStart, bEnd-bStart)
	if op {
		return abMat.T(), nil //opposite to icon. be ware
	}
	return abMat, nil
}

//GetResIdx : Calculate the resolution index based on the width of panel and the length of query region
func (e *HiC) GetResIdx(length int, width int) int {
	return e.getResIdx(length, width)
}
func (e *HiC) getResIdx(length int, width int) int {
	if width < 9 { //if width < 9 , width is resIdx
		return width
	}
	for i, v := range e.BpRes {
		if length/int(v) > width {
			if i == 0 {
				return 0
			} else {
				return i - 1
			}
		}
	}
	return 8
}

//QueryOne : get one matrix diag square submatrix
func (e *HiC) QueryOne(chrom string, start int, end int, width int) (mat64.Matrix, error) {
	return e.queryOne(bed3{chrom, start, end}, width)
}

//Icon : get the lowest resolution for one matrix
/*
func (e *HiC) Icon(chrom string) (mat64.Matrix, error) {
	idx := e.chr2idx(chrom)
	if idx == -1 {
		return nil, errors.New("chromosome not found")
	}
	key := strconv.Itoa(idx) + "_" + strconv.Itoa(idx)
	body, _ := e.loadBodyIndex(key)
	return body.Mats[0].Matrix().T(), nil
}

/* IconSmart : get the icon fit the width
*/
/*
func (e *HiC) IconSmart(chrom string, width int) (mat64.Matrix, error) {
	idx := e.chr2idx(chrom)
	if idx == -1 {
		return nil, errors.New("chromosome not found")
	}
	key := strconv.Itoa(idx) + "_" + strconv.Itoa(idx)
	body, _ := e.loadBodyIndex(key)
	for _, m := range body.Mats {
		r, _ := m.Dims()
		if r > width {
			return m.Matrix().T(), nil
		}
	}
	return body.Mats[8].Matrix().T(), nil
}
func (e *HiC) Icon2Smart(chrom string, chrom2 string, width int) (mat64.Matrix, error) { //chrom1 and chrom2 should be in order now.
	a := e.chr2idx(chrom)
	b := e.chr2idx(chrom2)
	if a == -1 || b == -1 {
		return nil, errors.New("chromosome not found")
	}
	op := false
	if a > b {
		op = true
		a, b = b, a
	}
	key := strconv.Itoa(a) + "_" + strconv.Itoa(b)
	body, _ := e.loadBodyIndex(key)
	for _, m := range body.Mats {
		_, c := m.Dims() //T()
		if c > width {
			if op {
				return m.Matrix(), nil
			} else {
				return m.Matrix().T(), nil
			}
		}
	}
	if op {
		return body.Mats[8].Matrix(), nil
	} else {
		return body.Mats[8].Matrix().T(), nil
	}
}
*/
func (e *HiC) queryOne(a bed3, width int) (mat64.Matrix, error) {
	resIdx := e.getResIdx(a.length(), width)
	return e._queryOne(a, resIdx)
}

/* Corrected: get correted bed position */
func (hic *HiC) Corrected(s int, e int, resIdx int) (int, int) {
	binsize := hic.BpRes[resIdx]
	start := (s / int(binsize)) * int(binsize)
	end := ((e-1)/int(binsize) + 1) * int(binsize)
	return start, end
}
func (e *HiC) _queryOne(a bed3, resIdx int) (mat64.Matrix, error) {
	binSize := e.BpRes[resIdx]
	start := a.start / int(binSize)
	end := (a.end-1)/int(binSize) + 1
	idx := e.chr2idx(a.chr)
	if idx == -1 {
		return nil, errors.New("chromosome not found")
	}
	key := strconv.Itoa(idx) + "_" + strconv.Itoa(idx)
	body, _ := e.loadBodyIndex(key)
	return body.Mats[resIdx].View(start, start, end-start, end-start), nil
}
