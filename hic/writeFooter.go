package hic

import (
	"io"

	"github.com/nimezhu/netio"
)

func (e *HiC) WriteIndex(w io.Writer) {
	netio.Write(w, e.Footer.NBytes)
	netio.Write(w, e.Footer.NEntrys)
	for k, v := range e.Footer.Entry {
		netio.WriteString(w, k)
		netio.Write(w, v.Position)
		netio.Write(w, v.Size)
	}
	nExpectedValues := int32(len(e.Footer.ExpectedValueMap))
	netio.Write(w, nExpectedValues)
	/*TODO
	for k, f := range e.Footer.ExpectedValueMap {
		netio.Write(w, int32(f.Unit())) //save int32 instead string
		netio.Write(w, int32(f.BinSize()))
	}
	*/
	/*
		for k, v := range e.Footer.NormVector {
		}
	*/
}

func (e *HiC) LoadIndex(r io.Reader) {

}
