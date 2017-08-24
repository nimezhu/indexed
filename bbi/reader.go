package bbi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
)

const BIGWIG_MAGIC = 0x888FFC26
const BIGBED_MAGIC = 0x8789F2EB

type BbiReader struct {
	Header    BbiHeader
	ChromData BData
	Index     RTree
	IndexZoom []RTree
	Fptr      io.ReadSeeker
	mutex     *sync.Mutex
}

func newBbiReader() *BbiReader {
	bwf := new(BbiReader)
	bwf.Header = *NewBbiHeader()
	bwf.ChromData = *NewBData()
	bwf.mutex = &sync.Mutex{}
	return bwf
}
func NewBbiReader(f io.ReadSeeker) *BbiReader {
	bwf := newBbiReader()
	bwf.Fptr = f
	return bwf
}
func (bwf *BbiReader) QueryRaw(idx, from, to int) <-chan *BbiBlockDecoderType { //Try to get raw data instead of binsize.
	ch := make(chan *BbiBlockDecoderType)
	log.Println("query raw data", idx, from, to)
	go func() {
		traverser := NewRTreeTraverser(&bwf.Index)
		defer close(ch)
		for r := range traverser.QueryVertices(idx, from, to) {
			block, err := r.Vertex.ReadBlockFromReader(bwf, r.Idx)
			if err != nil {
				log.Panic(err)
				return
			}
			decoder, err := NewBbiBlockDecoder(block)
			if err != nil {
				log.Panic(err)
				return
			}
			for record := range decoder.Decode() {
				//log.Println(record)
				if record.Idx != idx || record.From < from || record.To > to {
					continue
				}
				ch <- &record
			}
		}

	}()
	return ch
}
func (bwf *BbiReader) Close() error {
	return nil //TODO ReadSeekCloser
}
func (bwf *BbiReader) Query(idx, from, to, binsize int) <-chan *BbiQueryType {
	from = divIntDown(from, binsize) * binsize
	to = divIntUp(to, binsize) * binsize
	channel := make(chan *BbiQueryType)
	go func() {
		bwf.query(channel, idx, from, to, binsize)
		close(channel)
	}()
	return channel
}

func (bwf *BbiReader) query(channel chan *BbiQueryType, idx, from, to, binsize int) {
	// index of a matching zoom level for the given binsize
	zoomIdx := -1
	for i := 0; i < int(bwf.Header.ZoomLevels); i++ {
		//fmt.Println(bwf.Header.ZoomHeaders[i].ReductionLevel, "<", binsize)
		if binsize >= int(bwf.Header.ZoomHeaders[i].ReductionLevel) &&
			(binsize%int(bwf.Header.ZoomHeaders[i].ReductionLevel) == 0) {
			zoomIdx = i
		}
	}
	if zoomIdx != -1 {
		bwf.queryZoom(channel, zoomIdx, idx, from, to, binsize)
	} else {
		bwf.queryRaw(channel, idx, from, to, binsize)
	}
}

func (bwf *BbiReader) queryRaw(channel chan *BbiQueryType, idx, from, to, binsize int) {
	// no zoom level found, try raw data
	//log.Println("in query raw")
	c := make(chan bool, 1)
	go func() {
		traverser := NewRTreeTraverser(&bwf.Index)
		// current zoom record
		var result *BbiQueryType
		bwf.mutex.Lock()
		for r := range traverser.QueryVertices(idx, from, to) {
			block, err := r.Vertex.ReadBlockFromReader(bwf, r.Idx)
			if err != nil {
				channel <- &BbiQueryType{Error: err}
				return
			}
			decoder, err := NewBbiBlockDecoder(block)
			if err != nil {
				channel <- &BbiQueryType{Error: err}
				return
			}
			for record := range decoder.Decode() {
				if record.From < from || record.To > to {
					continue
				}
				if (record.To-record.From) < binsize || binsize%(record.To-record.From) == 0 { //TODO  add (record.To-record.From) < binsize || is this correct?
					// check if current result record is full
					// fmt.Println(result.Max)
					//DEBUG

					if result == nil || (result.To-result.From) >= binsize {
						if result != nil {
							channel <- result
						}
						result = NewBbiQueryType()
						result.ChromId = idx
						result.From = record.From
					}
					// add contents of current record to the resulting record
					//fmt.Println(record.Value)
					result.AddValue(record.Value)
					result.To = record.To
				} else {
					//TODO
					//channel <- &BbiQueryType{Error: fmt.Errorf("invalid binsize")}
					if result != nil {
						channel <- result
					}
					singleRecord := NewBbiQueryType()
					singleRecord.ChromId = idx
					singleRecord.From = record.From
					singleRecord.To = record.To
					singleRecord.AddValue(record.Value)
					channel <- singleRecord
					result = nil
					//return not return but skip large binsize data. TODO handle large region data.
				}
			}
		}

		if result != nil { //no record added. handle.
			channel <- result
		}
		bwf.mutex.Unlock()
		c <- true
	}()
	<-c
}

func (bwf *BbiReader) queryZoom(channel chan *BbiQueryType, zoomIdx, idx, from, to, binsize int) {
	//log.Println("query zoom", zoomIdx)
	c := make(chan bool, 1)
	go func() {
		bwf.mutex.Lock()
		traverser := NewRTreeTraverser(&bwf.IndexZoom[zoomIdx])
		// current zoom record
		var result *BbiQueryType

		for r := range traverser.QueryVertices(idx, from, to) {
			block, err := r.Vertex.ReadBlockFromReader(bwf, r.Idx)
			if err != nil {
				channel <- &BbiQueryType{Error: err}
				return
			}
			decoder := NewBbiZoomBlockDecoder(block)

			for record := range decoder.Decode() {
				//log.Println("debug", record.ChromId, record.From)
				if record.ChromId != idx || record.From < from || record.To > to {
					continue
				}
				if (record.To-record.From) < binsize || binsize%(record.To-record.From) == 0 {
					// check if current result record is full
					if result == nil || (result.To-result.From) >= binsize {
						if result != nil {
							channel <- result
						}
						result = NewBbiQueryType()
						result.ChromId = idx
						result.From = record.From
					}
					// add contents of current record to the resulting record
					result.AddRecord(record.BbiSummaryRecord)
					result.To = record.To
				} else {
					channel <- &BbiQueryType{Error: fmt.Errorf("invalid binsize")}
					return
				}
			}
		}
		if result != nil {
			channel <- result
		}
		bwf.mutex.Unlock()
		c <- true
	}()
	<-c
}

func (vertex *RVertex) ReadBlockFromReader(bwf *BbiReader, i int) ([]byte, error) { //should be inner function for lock
	var err error
	block := make([]byte, vertex.Sizes[i])
	if err = readseekerReadAt(bwf.Fptr, binary.LittleEndian, int64(vertex.DataOffset[i]), &block); err != nil {
		return nil, err
	}
	if bwf.Header.UncompressBufSize != 0 {
		if block, err = uncompressSlice(block); err != nil {
			return nil, err
		}
	}
	return block, nil
}

func readseekerReadAt(file io.ReadSeeker, order binary.ByteOrder, offset int64, data interface{}) error {
	currentPosition, _ := file.Seek(0, 1)
	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}
	if err := binary.Read(file, order, data); err != nil {
		return err
	}
	if _, err := file.Seek(currentPosition, 0); err != nil {
		return err
	}
	return nil
}

func (bwf *BbiReader) WriteIndex(f io.WriteSeeker) error { //TODO Fix to ReadSeekWriter write to buffers
	err := bwf.Header.Write(f)
	if err != nil {
		return err
	}
	err = bwf.ChromData.Write(f)
	if err != nil {
		return err
	}
	err = bwf.Index.Write(f)
	if err != nil {
		return err
	}
	for i := 0; i < int(bwf.Header.ZoomLevels); i++ {
		err = bwf.IndexZoom[i].Write(f)
		if err != nil {
			return err
		}
	}
	return nil
}
func (bwf *BbiReader) ReadIndex(f io.ReadSeeker) error {
	err := bwf.Header.Read(f)
	if err != nil {
		return err
	}
	err = bwf.ChromData.Read(f)
	if err != nil {
		return err
	}
	err = bwf.Index.Read(f)
	if err != nil {
		return err
	}
	bwf.IndexZoom = make([]RTree, bwf.Header.ZoomLevels)
	for i := 0; i < int(bwf.Header.ZoomLevels); i++ {
		err = bwf.IndexZoom[i].Read(f)
		if err != nil {
			return err
		}
	}
	return nil
}
func (bwf *BbiReader) Genome() ([]string, []int) {
	seqnames := make([]string, len(bwf.ChromData.Keys))
	lengths := make([]int, len(bwf.ChromData.Keys))

	for i := 0; i < len(bwf.ChromData.Keys); i++ {
		if len(bwf.ChromData.Values[i]) != 8 {
			fmt.Errorf("wrong file")
		}
		idx := int(binary.LittleEndian.Uint32(bwf.ChromData.Values[i][0:4]))
		if idx >= len(bwf.ChromData.Keys) {
			fmt.Errorf("wrong index")
		}
		seqnames[idx] = strings.TrimRight(string(bwf.ChromData.Keys[i]), "\x00")
		lengths[idx] = int(binary.LittleEndian.Uint32(bwf.ChromData.Values[i][4:8]))
	}
	return seqnames, lengths
}
func (bwf *BbiReader) InitIndex() error {
	fmt.Println("readHeader")
	if err := bwf.Header.Read(bwf.Fptr); err != nil {
		return err
	}
	if (bwf.Header.Magic != BIGWIG_MAGIC) && (bwf.Header.Magic != BIGBED_MAGIC) {
		log.Printf("not a bigwig or bigbed file;")
		return errors.New("not a BigWig file or BigBed file")
	}
	if _, err := bwf.Fptr.Seek(int64(bwf.Header.CtOffset), 0); err != nil {
		return err
	}
	if err := bwf.ChromData.Read(bwf.Fptr); err != nil {
		return err
	}
	if _, err := bwf.Fptr.Seek(int64(bwf.Header.IndexOffset), 0); err != nil {
		return err
	}
	if err := bwf.Index.Read(bwf.Fptr); err != nil {
		return err
	}

	bwf.IndexZoom = make([]RTree, bwf.Header.ZoomLevels)
	for i := 0; i < int(bwf.Header.ZoomLevels); i++ {
		if _, err := bwf.Fptr.Seek(int64(bwf.Header.ZoomHeaders[i].IndexOffset), 0); err != nil {
			fmt.Println("Read ZoomLevels", i)
			return err
		}
		if err := bwf.IndexZoom[i].Read(bwf.Fptr); err != nil {
			fmt.Println("Read Zoom", i)
			return err
		}
	}
	return nil
}
