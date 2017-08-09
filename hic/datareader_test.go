package hic

import (
	"testing"

	"fmt"

	. "github.com/nimezhu/netio"
)

const (
	testHic = "/home/zhuxp/Data/hic/test.hic"
)

func testReader(t *testing.T, uri string) {
	input, _ := NewReadSeeker(uri)
	a, _ := DataReader(input)
	t.Log(a)
	fmt.Println(a.Entrys())
	body, _ := a.loadBodyIndex("1_1")
	fmt.Println(body)
	blockMatrix := body.Mats[1]
	fmt.Println(blockMatrix.String())

	body.Mats[7].loadBlock(0)
	fmt.Println(body.Mats[7].buffers[0])
	fmt.Println("blockCount", body.Mats[4].BlockCount)
	fmt.Println("blockBinCount", body.Mats[4].BlockBinCount)
	fmt.Println(a.Footer.NormTypes)
	fmt.Println(a.Footer.ExpectedValueMap)
	fmt.Println("NormTypes:")
	n := a.Footer.NormTypeStrs()
	for i, k := range a.Footer.NormTypeStrings() {
		fmt.Println(i, n[i], k)
	}
	for k := range a.Footer.Units {
		fmt.Println("Unit:", k)
	}
	/*
		fmt.Println(body.Mats[4].coordsToBlockIndexes(820, 820, 30, 30))
		vm := body.Mats[4].View(820, 820, 30, 30)
		fmt.Println("At 1,1", body.Mats[1].At(1, 1))
		fmt.Println(sprintMat64(vm))
	*/
}

func TestLocalReader(t *testing.T) {
	uri := testHic
	t.Log("testing uri :", uri)
	testReader(t, uri)
}

func TestQuery(t *testing.T) {
	uri := testHic
	input, _ := NewReadSeeker(uri)
	a, _ := DataReader(input)
	fmt.Println(a)
	b, _ := a.queryOne(bed3{"1", 10000000, 20000000}, 10)
	fmt.Println(sprintMat64(b))

	m, _ := a.queryTwo(bed3{"1", 10000000, 20000000}, bed3{"2", 10000000, 15000000}, 15)
	for _, v := range m {
		fmt.Println(sprintMat64(v))
	}

	bed := bed3{"2", 0, 2000000}
	resIdx := 3
	vec, _ := a.queryVector(bed, 0, 0, resIdx)
	fmt.Println("vector 0,0,2", vec)

	mat, _ := a.queryOneNormMat(bed, 0, 0, resIdx)
	fmt.Println("NORM")
	fmt.Println(sprintMat64(mat))
	raw, _ := a._queryOne(bed, resIdx)
	fmt.Println("RAW")
	fmt.Println(sprintMat64(raw))

	matE, _ := a.queryOneFoldChangeOverExpected(bed, 0, 0, resIdx)
	fmt.Println("FOLDCHANGE")
	fmt.Println(sprintMat64(matE))
	bed2 := bed3{"1", 1000000, 2000000}
	vec2, _ := a.queryVector(bed2, 0, 0, resIdx)
	fmt.Println("vector B", vec2)
	mat12, _ := a.queryTwoNormMat(bed, bed2, 0, 0, resIdx)
	fmt.Println("NORM TWO")
	fmt.Println(sprintMat64(mat12))
	raw12, _ := a._queryTwo(bed, bed2, resIdx)
	fmt.Println("RAW TWO")
	fmt.Println(sprintMat64(raw12))
}
func TestGetExpect(t *testing.T) {
	uri := testHic
	input, _ := NewReadSeeker(uri)
	a, _ := DataReader(input)
	// unitIdx + binsize + normIdx
	e, ok := a.Footer.ExpectedValueMap["0_2500000_0"]
	if ok {
		fmt.Println("expected value ", e)
	}
}
func TestAllChr(t *testing.T) {
	uri := testHic
	input, _ := NewReadSeeker(uri)
	a, _ := DataReader(input)
	b, _ := a.loadBodyIndex("0_0")
	b0 := b.Mats[0].Matrix()
	fmt.Println(b.Mats[0].String())
	fmt.Println(b0.Dims())
	fmt.Println(sprintMat64(b.Mats[0].View(0, 0, 10, 10)))
}
