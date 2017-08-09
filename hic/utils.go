package hic

import (
	"bytes"
	"fmt"

	"github.com/gonum/matrix/mat64"
)

func sprintMat64(t mat64.Matrix) string {
	r, c := t.Dims()
	var buffer bytes.Buffer
	for i := 0; i < r; i++ {
		buffer.WriteString(fmt.Sprintf("%.2f", t.At(i, 0)))
		for j := 1; j < c; j++ {
			buffer.WriteString(fmt.Sprintf("\t%.2f", t.At(i, j)))
		}
		buffer.WriteString("\n")
	}
	return buffer.String()
}
