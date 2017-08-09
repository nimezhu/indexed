package hic

import (
	"bytes"
	"fmt"
)

type Body struct {
	Chr1Idx int32
	Chr2Idx int32
	NRes    int32
	Mats    []BlockMatrix //map resolution to blockmatrix
}

func (b *Body) String() string {
	var s bytes.Buffer
	s.WriteString("Body\n")
	s.WriteString(fmt.Sprintf("\tchr1Idx\t%d\n", b.Chr1Idx))
	s.WriteString(fmt.Sprintf("\tchr2Idx\t%d\n", b.Chr2Idx))
	s.WriteString(fmt.Sprintf("\tnResolutions\t%d\n", b.NRes))
	return s.String()
}
