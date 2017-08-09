package hic

import (
	"bytes"
	"errors"
	"strconv"

	. "github.com/nimezhu/netio"
)

type NormVector struct {
	normType   int
	chrIdx     int
	unit       int
	resolution int
	index      *Index
	loaded     bool
	data       []float64
}

func NewNormVector(normType int, chrIdx int, unit int, resolution int, index *Index) *NormVector {
	return &NormVector{normType, chrIdx, unit, resolution, index, false, []float64{}}
}
func (v *NormVector) Key() string {
	return normalizationVectorGetKey(v.normType, v.chrIdx, v.unit, v.resolution)
}
func (v *NormVector) Data() ([]float64, error) {
	if v.loaded {
		return v.data, nil
	}
	return []float64{}, errors.New("loaded data from file first")
}
func (v *NormVector) Load(reader MutexReadSeeker) {
	if v.loaded {
		return
	}
	reader.Lock()
	b := make([]byte, v.index.Size)
	reader.Seek(v.index.Position, 0)
	reader.Read(b)
	br := bytes.NewReader(b)
	l, _ := ReadInt(br)
	v.data = make([]float64, l)
	for i := int32(0); i < l; i++ {
		v.data[i], _ = ReadFloat64(br)
		//fmt.Println(i, v.data[i])
	}
	reader.Unlock()
	v.loaded = true
	return
}
func normalizationVectorGetKey(normType int, chrIdx int, unit int, resolution int) string {
	t := strconv.Itoa(normType) //normtype
	c := strconv.Itoa(chrIdx)
	u := strconv.Itoa(unit) //BP 0, FRAG 1
	bin := strconv.Itoa(resolution)
	return t + "_" + c + "_" + u + "_" + bin
}
