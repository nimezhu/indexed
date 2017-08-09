package hic

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/nimezhu/indexed/hic/normtype"
	"github.com/nimezhu/indexed/hic/unit"
)

type Footer struct {
	NBytes           int32
	NEntrys          int32
	Entry            map[string]Index
	ExpectedValueMap map[string]*ExpectedValueFunc //expected value map string format: unitIdx + "_" + binsize + "_" + normTypeIdx
	NormVector       map[string]*NormVector
	NormTypes        map[int]bool
	Units            map[int]bool
}

func (f *Footer) String() string {
	var s bytes.Buffer
	s.WriteString("Footer\n")
	s.WriteString(fmt.Sprintf("\tNBytes\t%d\n", f.NBytes))
	s.WriteString(fmt.Sprintf("\tNEntries\t%d\n", f.NEntrys))
	s.WriteString("\tNormTypes: ")
	for v := range f.NormTypes {
		s.WriteString(" ")
		s.WriteString(normtype.IdxToStr(v))
	}
	s.WriteString("\n\tUnits: ")
	for v := range f.Units {
		s.WriteString(" ")
		s.WriteString(unit.IdxToString(v))
	}

	s.WriteString("\n")
	return s.String()
}

/*NormTypeIdx: get all normalization types indexes
 */
func (f *Footer) NormTypeIdx() []int {
	var keys []int
	for k := range f.NormTypes {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

/*NormTypeStrings: get all norm type strings
 */
func (f *Footer) NormTypeStrings() []string {
	var keys []string
	for _, k := range f.NormTypeIdx() {
		keys = append(keys, normtype.IdxToString(k))
	}
	return keys
}

/*NormTypeStrs: get all norm type short strings
 */
func (f *Footer) NormTypeStrs() []string {
	var keys []string
	for _, k := range f.NormTypeIdx() {
		keys = append(keys, normtype.IdxToStr(k))
	}
	return keys
}
