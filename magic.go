package indexed

import (
	"encoding/binary"

	"github.com/nimezhu/netio"
)

const BIGWIG_MAGIC = 0x888FFC26
const BIGBED_MAGIC = 0x8789F2EB
const HIC_MAGIC = 0x00434948

func Magic(uri string) (string, error) {
	f, err := netio.NewReadSeeker(uri)
	if err != nil {
		return "unknown", err
	}
	p := make([]byte, 4)
	f.Read(p)
	n := binary.LittleEndian.Uint32(p)
	switch n {
	case BIGBED_MAGIC:
		return "bigbed", nil
	case BIGWIG_MAGIC:
		return "bigwig", nil
	case HIC_MAGIC:
		return "hic", nil
	default:
		return "unknown", nil

	}
	return "unknown", nil
}
