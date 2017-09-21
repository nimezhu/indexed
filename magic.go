package indexed

import (
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/nimezhu/netio"
)

const BIGWIG_MAGIC = 0x888FFC26
const BIGBED_MAGIC = 0x8789F2EB
const HIC_MAGIC = 0x00434948
const BIGSIZE = 100000000 //100Mb is bigbedLarge

func Magic(uri string) (string, error) {
	if _, err := os.Stat(filepath.Join(filepath.Dir(uri), "images")); err == nil {
		if _, err := os.Stat(uri + ".tbi"); err == nil {
			return "image", err
		}
	}
	f, err := netio.NewReadSeeker(uri)
	if err != nil {
		return "unknown", err
	}
	size, err := netio.Size(uri)
	if err != nil {
		return "unknown", err
	}
	p := make([]byte, 4)
	f.Read(p)
	n := binary.LittleEndian.Uint32(p)
	switch n {
	case BIGBED_MAGIC:
		if size > BIGSIZE {
			return "bigbedLarge", nil
		}
		return "bigbed", nil
	case BIGWIG_MAGIC:
		return "bigwig", nil
	case HIC_MAGIC:
		return "hic", nil
	}
	if _, err := os.Stat(uri + ".tbi"); err == nil {
		return "tabix", err
	}
	return "unknown", nil
}
