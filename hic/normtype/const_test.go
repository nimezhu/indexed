package normtype

import "testing"

func TestConst(t *testing.T) {
	a := StringToIdx("None")
	t.Log(a)
	a1 := StringToIdx("VC")
	t.Log(a1)
	b := StringToIdx("Balanced")
	t.Log(b)
	c := IdxToString(5)
	t.Log(c)
}
