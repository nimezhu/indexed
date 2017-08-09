package hic

import (
	"errors"
	"io"
	"sync"

	. "github.com/nimezhu/netio"
)

func DataReader(buf io.ReadSeeker) (*HiC, error) {
	hic := NewHiC()
	hic.mutex = &sync.Mutex{}
	hic.Reader = buf
	magic, _ := ReadString(buf)
	if magic != MAGIC {
		return nil, errors.New("not a HiC format file")
	}
	hic.Version, _ = ReadInt(buf)
	hic.masterIndexPos, _ = ReadLong(buf)
	hic.Genome, _ = ReadString(buf)

	if hic.Version > 4 {
		hic.NAttr, _ = ReadInt(buf)
		//fmt.Println("n:", nAttrbutes)
		hic.Attr = make(map[string]string)
		for i := 0; i < int(hic.NAttr); i++ {
			key, _ := ReadString(buf)
			value, _ := ReadString(buf)
			hic.Attr[key] = value
		}
	}

	hic.NChrs, _ = ReadInt(buf)
	hic.Chr = make([]Chr, hic.NChrs)
	for i := 0; i < int(hic.NChrs); i++ {
		chr, _ := ReadString(buf)
		length, _ := ReadInt(buf)
		hic.Chr[i] = Chr{chr, length}

	}

	hic.NBpRes, _ = ReadInt(buf)
	hic.BpRes = make([]int32, hic.NBpRes)
	for i := int32(0); i < hic.NBpRes; i++ {
		hic.BpRes[i], _ = ReadInt(buf)

	}
	hic.NFragRes, _ = ReadInt(buf)
	hic.FragRes = make([]int32, hic.NFragRes)
	for i := int32(0); i < hic.NFragRes; i++ {
		hic.FragRes[i], _ = ReadInt(buf)
	}
	//BODY
	//FOOTER
	hic.readFooter()
	return &hic, nil
}
