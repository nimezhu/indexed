package indexed

import (
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/nimezhu/netio"
)

const BIGWIG_MAGIC = 0x888FFC26
const BIGBED_MAGIC = 0x8789F2EB
const TWOBIT_MAGIC = 0x1A412743
const HIC_MAGIC = 0x00434948

var GZIP_MAGIC = []byte("\x1f\x8b")

const BIGSIZE = 100000000 //100Mb is bigbedLarge
func MagicReadSeeker(f io.ReadSeeker) (string, error) {
	p := make([]byte, 4)
	f.Seek(0, 0)
	defer f.Seek(0, 0)
	l, err := f.Read(p)
	if p[0] == GZIP_MAGIC[0] && p[1] == GZIP_MAGIC[1] {
		return "gzip", nil
	}
	if err != nil {
		log.Println(l, err)
	}
	n := binary.LittleEndian.Uint32(p)
	switch n {
	case BIGBED_MAGIC:
		return "bigbed", nil
	case BIGWIG_MAGIC:
		return "bigwig", nil
	case HIC_MAGIC:
		return "hic", nil
	case TWOBIT_MAGIC:
		return "twobit", nil
	}
	return "unknown", errors.New("unknown format")
}
func Magic(uri string) (string, error) {
	if _, err := os.Stat(filepath.Join(filepath.Dir(uri), "images")); err == nil {
		if _, err := os.Stat(uri + ".tbi"); err == nil {
			return "image", err
		}
	}
	if _, err := os.Stat(uri + ".tbi"); err == nil {
		return "tabix", err
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
	if p[0] == GZIP_MAGIC[0] && p[1] == GZIP_MAGIC[1] {
		return "gzip", nil
	}
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
	case TWOBIT_MAGIC:
		return "twobit", nil
	}

	return "unknown", nil
}
