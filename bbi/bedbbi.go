package bigwig

import (
	"bytes"
	"fmt"

	"github.com/nimezhu/netio"
)

type BedBbiDataHeader struct {
	ChromId uint32
	Start   uint32
	End     uint32
}
type BedBbiBlockDecoder struct {
	//Header BedBbiDataHeader
	Buffer []byte
}
type BedBbiBlockDecoderType struct {
	Idx   int
	From  int
	To    int
	Value string
	Error error
}

func NewBedBbiBlockDecoder(buffer []byte) (*BedBbiBlockDecoder, error) {
	if len(buffer) < 0 {
		return nil, fmt.Errorf("block length is shorter than 12 bytes")
	}
	//header := BedBbiDataHeader{}
	//header.ChromId = binary.LittleEndian.Uint32(buffer[0:4])
	//header.Start = binary.LittleEndian.Uint32(buffer[4:8])
	//header.End = binary.LittleEndian.Uint32(buffer[8:12])
	reader := BedBbiBlockDecoder{}
	//reader.Header = header
	reader.Buffer = buffer
	return &reader, nil

}

func (reader *BedBbiBlockDecoder) fillChannel(channel chan *BedBbiBlockDecoderType) {
	a := bytes.NewReader(reader.Buffer)
	for true {
		r := BedBbiBlockDecoderType{}
		idx, err := netio.ReadInt(a)
		if err != nil {
			//log.Println("should be EOF?", err)
			break
		}
		from, _ := netio.ReadInt(a)
		to, _ := netio.ReadInt(a)
		value, _ := netio.ReadString(a)
		//
		//fmt.Println("debug", idx, from, to) //TODO
		r.Idx = int(idx) //TO TEST
		r.From = int(from)
		r.To = int(to)
		r.Value = value
		channel <- &r
	}
}

func (reader *BedBbiBlockDecoder) Decode() <-chan *BedBbiBlockDecoderType {
	channel := make(chan *BedBbiBlockDecoderType)
	go func() {
		reader.fillChannel(channel)
		close(channel)
	}()
	return channel
}

/*

func (reader *BedBbiBlockDecoder) importRecord(s []float64, n []int, binsize int, record *BedBbiBlockDecoderType) error {
	for i := record.From / binsize; i < record.To/binsize; i++ {
		if i < len(s) {
			s[i] = (float64(n[i])*s[i] + 1.0) / (float64(n[i]) + 1.0)
			n[i] += 1
		} else {
			return fmt.Errorf("position `%d' is out of range (trying to access bin `%d' but sequence has only `%d' bins)", i*binsize, i, len(s))
		}
	}
	return nil
}
*/
//TODO.
/*
func (reader *BedBbiBlockDecoder) Import(sequence []float64, binsize int) error {
	r := &BedBbiBlockDecoderType{}
	n := make([]int, len(sequence))
	//TODO
	fmt.Println("TODO", r, n)
	return nil
}
*/

func (bwf *BbiReader) QueryBedRaw(idx, from, to int) <-chan *BedBbiBlockDecoderType {
	channel := make(chan *BedBbiBlockDecoderType)
	go func() {
		bwf.queryBedRaw(channel, idx, from, to)
		close(channel)
	}()
	return channel
}

/*
func (bwf *BbiReader) queryBed(channel chan *BedBbiBlockDecoderType, idx, from, to int) {
	// index of a matching zoom level for the given binsiz
	bwf.queryRawBed(channel, idx, from, to)
}
*/
func (bwf *BbiReader) queryBedRaw(channel chan *BedBbiBlockDecoderType, idx, from, to int) {
	// no zoom level found, try raw data
	traverser := NewRTreeTraverser(&bwf.Index)
	// current zoom record

	for r := range traverser.QueryVertices(idx, from, to) {
		block, err := r.Vertex.ReadBlockX7(bwf, r.Idx)
		if err != nil {
			channel <- &BedBbiBlockDecoderType{Error: err}
			return
		}
		decoder, err := NewBedBbiBlockDecoder(block)
		if err != nil {
			channel <- &BedBbiBlockDecoderType{Error: err}
			return
		}
		for record := range decoder.Decode() {
			if record.To < from || record.From > to {
				continue
			}
			channel <- record
		}
	}
}
