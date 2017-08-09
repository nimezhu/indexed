package unit

import "strings"

//type NormType int

const (
	BP = iota
	FRAG
)

var (
	idx2strings = []string{
		"BP",
		"FRAG",
	}
)

func IdxToString(i int) string {
	switch {
	case i > len(idx2strings):
		return "None"
	case i < 0:
		return "None"
	default:
		return idx2strings[i]
	}
}
func StringToIdx(s string) int {
	switch strings.ToLower(s) {
	case "bp":
		return BP
	case "frag":
		return FRAG
	}
	return BP
}
