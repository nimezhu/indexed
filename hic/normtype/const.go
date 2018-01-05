package normtype

import "strings"

//type NormType int

const (
	NONE = iota
	VC
	VC_SQRT
	KR
	GW_KR
	INTER_KR
	GW_VC
	INTER_VC
	LOADED
)

var (
	idx2strings = []string{
		"None",
		"Coverage",
		"Coverage (Sqrt)",
		"Balanced",
		"Genome-Wide Balanced",
		"Inter Balanced",
		"Genome-Wide Coverage",
		"Inter Coverage",
		"Loaded",
	}
	idx2strs = []string{
		"NONE",
		"VC",
		"VC_SQRT",
		"KR",
		"GW_KR",
		"INTER_KR",
		"GW_VC",
		"INTER_VC",
		"LOADED",
	}
)

func IdxToStr(i int) string {
	switch {
	case i > len(idx2strs):
		return "NONE"
	case i < 0:
		return "NONE"
	default:
		return idx2strs[i]
	}
}
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
	case "none":
		return NONE
	case "coverage", "vc":
		return VC
	case "coverage (sqrt)", "vc_sqrt":
		return VC_SQRT
	case "balanced", "kr":
		return KR
	case "genome-wide balanced", "gw_kr":
		return GW_KR
	case "inter balanced", "inter_kr":
		return INTER_KR
	case "genome-wide coverage", "gw_vc":
		return GW_VC
	case "inter coverage", "inter_vc":
		return INTER_VC
	case "loaded":
		return LOADED
	}
	return NONE
}
