package bbi

import (
	"errors"
	"log"
)

type Chromo struct {
	Idx    int
	Name   string
	Length int
}
type Genome struct {
	Chrs    []Chromo
	Chr2Idx map[string]int
}

type BigWigReader struct {
	Reader *BbiReader
	Genome Genome
}

func NewBigWigReader(b *BbiReader) *BigWigReader {
	bw := BigWigReader{}
	chroms, lengths := b.Genome()
	l := len(chroms)
	bw.Reader = b
	bw.Genome.Chrs = make([]Chromo, l)
	bw.Genome.Chr2Idx = make(map[string]int)
	for i := 0; i < l; i++ {
		bw.Genome.Chrs[i] = Chromo{i, chroms[i], lengths[i]}
		bw.Genome.Chr2Idx[chroms[i]] = i
	}
	return &bw
}

var minNumValues = 30

func (bw *BigWigReader) GetBinsize(length int, width int) int {
	var binsize int
	if length == 0 {
		return 1
	}
	if width/(length/int(bw.Reader.Header.ZoomHeaders[0].ReductionLevel)) > minNumValues {
		//return -1 //query raw data then. no return -1 this time try original.

		var b = length / width
		if b > 0 {
			return b
		} else {
			return 1
		}
	}
	last := length / width
	for i := uint16(0); i < bw.Reader.Header.ZoomLevels; i++ {
		binsize = int(bw.Reader.Header.ZoomHeaders[i].ReductionLevel)
		if width > length/binsize {
			break
		}
		last = binsize
	}
	return last
}
func (bw *BigWigReader) Binsizes() []int {
	binsizes := []int{}
	for i := uint16(0); i < bw.Reader.Header.ZoomLevels; i++ {
		binsizes = append(binsizes, int(bw.Reader.Header.ZoomHeaders[i].ReductionLevel))
	}
	return binsizes
}
func (bw *BigWigReader) Query(chr string, start int, end int, width int) (<-chan *BbiQueryType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		log.Printf("chromsome not found")
		return nil, errors.New("chromosome not found") //TODO get error
	}
	binsize := bw.GetBinsize(end-start, width)
	return bw.Reader.Query(idx, start, end, binsize), nil
}
func (bw *BigWigReader) QueryRaw(chr string, start int, end int) (<-chan *BbiBlockDecoderType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		log.Printf("chromsome not found")
		return nil, errors.New("chromosome not found") //TODO get error
	}
	return bw.Reader.QueryRaw(idx, start, end), nil
}

/* QueryBin:
 * Query with binsize
 */
func (bw *BigWigReader) QueryBin(chr string, start int, end int, binsize int) (<-chan *BbiQueryType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		return nil, errors.New("chromosome not found") //TODO get error
	}
	return bw.Reader.Query(idx, start, end, binsize), nil
}
