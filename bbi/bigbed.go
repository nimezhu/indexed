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
func (bw *BigBedReader) getBinsize(length int, width int) int {
	var binsize int
	if length == 0 {
		return 1
	}
	if bw.Reader.Header.ZoomLevels == 0 {
		var b = length / width
		if b > 0 {
			return b
		} else {
			return 1
		}
	}
	if width/(length/int(bw.Reader.Header.ZoomHeaders[0].ReductionLevel)) > minNumValues {
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

func (bw *BigBedReader) Query(chr string, start int, end int, width int) (<-chan *BedBbiQueryType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		log.Printf("chromsome not found")
		return nil, errors.New("chromosome not found") //TODO get error
	}
	binsize := bw.getBinsize(end-start, width)
	return bw.Reader.QueryBedBin(idx, start, end, binsize), nil
}

func (bw *BigBedReader) QueryRaw(chr string, start int, end int) (<-chan *BedBbiBlockDecoderType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		log.Printf("chromsome not found")
		return nil, errors.New("chromosome not found") //TODO get error
	}
	return bw.Reader.QueryBedRaw(idx, start, end), nil
}

func (bw *BigBedReader) QueryBin(chr string, start int, end int, binsize int) (<-chan *BedBbiQueryType, error) {
	idx, ok := bw.Genome.Chr2Idx[chr]
	if !ok {
		return nil, errors.New("chromosome not found") //TODO get error
	}
	return bw.Reader.QueryBedBin(idx, start, end, binsize), nil
}
