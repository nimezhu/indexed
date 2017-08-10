package bbi

import (
	"errors"
	"fmt"
	"log"
)

type BigBedReader struct {
	Reader *BbiReader
	Genome Genome
}

func NewBigBedReader(b *BbiReader) *BigBedReader {
	bb := BigBedReader{}
	chroms, lengths := b.Genome()
	l := len(chroms)
	bb.Reader = b
	bb.Genome.Chrs = make([]Chromo, l)
	bb.Genome.Chr2Idx = make(map[string]int)
	for i := 0; i < l; i++ {
		bb.Genome.Chrs[i] = Chromo{i, chroms[i], lengths[i]}
		bb.Genome.Chr2Idx[chroms[i]] = i
	}
	return &bb
}
func (bb *BigBedReader) Format(e *BedBbiBlockDecoderType) string {
	chr := bb.Genome.Chrs[e.Idx].Name
	s := fmt.Sprintf("%s\t%d\t%d\t%s", chr, e.From, e.To, e.Value)
	return s
}

func (bw *BigBedReader) Binsizes() []int {
	binsizes := []int{}
	for i := uint16(0); i < bw.Reader.Header.ZoomLevels; i++ {
		binsizes = append(binsizes, int(bw.Reader.Header.ZoomHeaders[i].ReductionLevel))
	}
	return binsizes
}

/*
func (bw *BigBedReader) Query(chr string, start int, end int, width int) (<-chan *BedBbiQueryType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		log.Printf("chromsome not found")
		return nil, errors.New("chromosome not found") //TODO get error
	}
	binsize := bw.GetBinsize(end-start, width)
	return bw.Reader.Query(idx, start, end, binsize), nil
}
*/
func (bw *BigBedReader) QueryRaw(chr string, start int, end int) (<-chan *BedBbiBlockDecoderType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		log.Printf("chromsome not found")
		return nil, errors.New("chromosome not found") //TODO get error
	}
	return bw.Reader.QueryBedRaw(idx, start, end), nil
}

/* QueryBin: TODO.
 * Query with binsize
 */
/*
func (bw *BigBedReader) QueryBin(chr string, start int, end int, binsize int) (<-chan *BbiQueryType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		return nil, errors.New("chromosome not found") //TODO get error
	}
	return bw.Reader.Query(idx, start, end, binsize), nil
}
*/
