/* Copyright (C) 2016 Philipp Benner
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

/*　add support to io.ReadSeeker interface by Xiaopeng Zhu
 *
 */
package bbi

/* -------------------------------------------------------------------------- */

// Methods for reading and writing Big Binary Indexed files such as bigWig
// and bigBed

/* -------------------------------------------------------------------------- */

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/nimezhu/netio"
)

import "compress/zlib"

import "math"

import "io"

/* -------------------------------------------------------------------------- */

const CIRTREE_MAGIC = 0x78ca8c91
const IDX_MAGIC = 0x2468ace0

const BbiMaxZoomLevels = 10 /* Max number of zoom levels */
const BbiResIncrement = 4   /* Amount to reduce at each zoom level */

/* -------------------------------------------------------------------------- */

func fileReadAt(file io.ReadSeeker, order binary.ByteOrder, offset int64, data interface{}) error {
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

func fileWriteAt(file io.WriteSeeker, order binary.ByteOrder, offset int64, data interface{}) error {
	currentPosition, _ := file.Seek(0, 1)
	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}
	if err := binary.Write(file, order, data); err != nil {
		return err
	}
	if _, err := file.Seek(currentPosition, 0); err != nil {
		return err
	}
	return nil
}

func uncompressSlice(data []byte) ([]byte, error) {
	b := bytes.NewReader(data)
	z, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer z.Close()

	return ioutil.ReadAll(z)
}

func compressSlice(data []byte) ([]byte, error) {
	var b bytes.Buffer
	z, err := zlib.NewWriterLevel(&b, zlib.BestCompression)
	if err != nil {
		//panic(err)
		return nil, err
	}
	_, err = z.Write(data)
	if err != nil {
		return nil, err
	}
	z.Close()

	return b.Bytes(), nil
}

/* -------------------------------------------------------------------------- */

type BbiZoomRecord struct {
	ChromId    uint32
	Start      uint32
	End        uint32
	Valid      uint32
	Min        float32
	Max        float32
	Sum        float32
	SumSquares float32
}

func (record *BbiZoomRecord) Read(reader io.Reader) error {
	if err := binary.Read(reader, binary.LittleEndian, &record.ChromId); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.Start); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.End); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.Valid); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.Min); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.Max); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.Sum); err != nil {
		return err
	}
	if err := binary.Read(reader, binary.LittleEndian, &record.SumSquares); err != nil {
		return err
	}
	return nil
}

func (record BbiZoomRecord) Write(writer io.Writer) error {
	if err := binary.Write(writer, binary.LittleEndian, record.ChromId); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.Start); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.End); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.Valid); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.Min); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.Max); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.Sum); err != nil {
		return err
	}
	if err := binary.Write(writer, binary.LittleEndian, record.SumSquares); err != nil {
		return err
	}
	return nil
}

/* -------------------------------------------------------------------------- */

type BbiSummaryRecord struct {
	ChromId    int
	From       int
	To         int
	Valid      int
	Min        float64
	Max        float64
	Sum        float64
	SumSquares float64
}

func NewBbiSummaryRecord() BbiSummaryRecord {
	record := BbiSummaryRecord{}
	record.Min = math.Inf(1)
	record.Max = math.Inf(-1)
	return record
}

func (record *BbiSummaryRecord) AddRecord(x BbiSummaryRecord) {
	record.Valid += x.Valid
	record.Min = math.Min(record.Min, x.Min)
	record.Max = math.Max(record.Max, x.Max)
	record.Sum += x.Sum
	record.SumSquares += x.SumSquares
}

func (record *BbiSummaryRecord) AddValue(x float64) {
	record.Valid += 1
	record.Min = math.Min(record.Min, x)
	record.Max = math.Max(record.Max, x)
	record.Sum += x
	record.SumSquares += x * x
}

/* -------------------------------------------------------------------------- */

type BbiBlockDecoder struct {
	Header BbiDataHeader
	Buffer []byte
}

type BbiBlockDecoderType struct {
	Idx   int
	From  int
	To    int
	Value float64
	Error error
}

func NewBbiBlockDecoder(buffer []byte) (*BbiBlockDecoder, error) {
	if len(buffer) < 24 {
		return nil, fmt.Errorf("block length is shorter than 24 bytes")
	}
	reader := BbiBlockDecoder{}
	// parse header
	reader.Header.ReadBuffer(buffer)
	// crop header from buffer
	reader.Buffer = buffer[24:]
	//fmt.Println("header type", reader.Header.Type)
	//fmt.Println("buffer", reader.Buffer)
	switch reader.Header.Type {
	/*
		default:
			return nil, fmt.Errorf("unsupported block type")
	*/
	/*
		case 0:
			if true {
				return &reader, nil
			}
			//return nil, fmt.Errorf("big bed")
	*/
	case 1:
		if len(reader.Buffer)%12 != 0 {
			return nil, fmt.Errorf("bedGraph data block has invalid length")
		}
	case 2:
		if len(buffer)%8 != 0 {
			return nil, fmt.Errorf("variable step data block has invalid length")
		}
	case 3:
		if len(buffer)%4 != 0 {
			return nil, fmt.Errorf("fixed step data block has invalid length")
		}
	default: //BigBed?
		reader.Buffer = buffer[12:]
		return &reader, nil
	}
	return &reader, nil
}

func (reader *BbiBlockDecoder) fillChannel(channel chan BbiBlockDecoderType) {
	switch reader.Header.Type {
	/*
		default:
			// this shouldn't happen
			panic("internal error (unsupported block type)")
	*/
	case 1:
		for i := 0; i < len(reader.Buffer); i += 12 {
			r := BbiBlockDecoderType{}
			r.Idx = i
			r.From = int(binary.LittleEndian.Uint32(reader.Buffer[i+0 : i+4]))
			r.To = int(binary.LittleEndian.Uint32(reader.Buffer[i+4 : i+8]))
			r.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(reader.Buffer[i+8 : i+12])))
			channel <- r
		}
	case 2:
		for i := 0; i < len(reader.Buffer); i += 8 {
			r := BbiBlockDecoderType{}
			r.Idx = i
			r.From = int(binary.LittleEndian.Uint32(reader.Buffer[i+0 : i+4]))
			r.To = r.From + int(reader.Header.Span)
			r.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(reader.Buffer[i+4 : i+8])))
			channel <- r
		}
	case 3:
		for i := 0; i < len(reader.Buffer); i += 4 {
			r := BbiBlockDecoderType{}
			r.Idx = i
			r.From = int(reader.Header.Start + uint32(i/4)*reader.Header.Step)
			r.To = r.From + int(reader.Header.Span)
			r.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(reader.Buffer[i : i+4])))
			channel <- r
		}
	default: //BIGBED OR BIGTAB?
		a := bytes.NewReader(reader.Buffer)
		for true {
			r := BbiBlockDecoderType{}
			value, err := netio.ReadString(a)
			if err != nil {
				break
			}
			idx, _ := netio.ReadInt(a)
			from, _ := netio.ReadInt(a)
			to, _ := netio.ReadInt(a)
			fmt.Println(idx, from, to, value)
			r.Idx = int(idx) //TO TEST
			r.From = int(from)
			r.To = int(to)
			r.Value = 0.0
			channel <- r
		}
	}

}

func (reader *BbiBlockDecoder) Decode() <-chan BbiBlockDecoderType {
	channel := make(chan BbiBlockDecoderType)
	go func() {
		reader.fillChannel(channel)
		close(channel)
	}()
	return channel
}

func (reader *BbiBlockDecoder) importRecord(s []float64, n []int, binsize int, record *BbiBlockDecoderType) error {
	for i := record.From / binsize; i < record.To/binsize; i++ {
		if i < len(s) {
			s[i] = (float64(n[i])*s[i] + record.Value) / (float64(n[i]) + 1.0)
			n[i] += 1
		} else {
			return fmt.Errorf("position `%d' is out of range (trying to access bin `%d' but sequence has only `%d' bins)", i*binsize, i, len(s))
		}
	}
	return nil
}

func (reader *BbiBlockDecoder) Import(sequence []float64, binsize int) error {
	if binsize > int(reader.Header.Span) && binsize%int(reader.Header.Span) != 0 ||
		(int(reader.Header.Span) > binsize && int(reader.Header.Span)%binsize != 0) {
		return fmt.Errorf("cannot import data with binsize `%d' into track with binsize `%d'", reader.Header.Span, binsize)
	}
	r := &BbiBlockDecoderType{}
	n := make([]int, len(sequence))
	switch reader.Header.Type {
	default:
		return fmt.Errorf("unsupported block type")
	case 1:
		for i := 0; i < len(reader.Buffer); i += 12 {
			r.Idx = i
			r.From = int(binary.LittleEndian.Uint32(reader.Buffer[i+0 : i+4]))
			r.To = int(binary.LittleEndian.Uint32(reader.Buffer[i+4 : i+8]))
			r.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(reader.Buffer[i+8 : i+12])))
			if err := reader.importRecord(sequence, n, binsize, r); err != nil {
				return err
			}
		}
	case 2:
		for i := 0; i < len(reader.Buffer); i += 8 {
			r.Idx = i
			r.From = int(binary.LittleEndian.Uint32(reader.Buffer[i+0 : i+4]))
			r.To = r.From + int(reader.Header.Span)
			r.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(reader.Buffer[i+4 : i+8])))
			if err := reader.importRecord(sequence, n, binsize, r); err != nil {
				return err
			}
		}
	case 3:
		for i := 0; i < len(reader.Buffer); i += 4 {
			r.Idx = i
			r.From = int(reader.Header.Start + uint32(i/4)*reader.Header.Step)
			r.To = r.From + int(reader.Header.Span)
			r.Value = float64(math.Float32frombits(binary.LittleEndian.Uint32(reader.Buffer[i : i+4])))
			if err := reader.importRecord(sequence, n, binsize, r); err != nil {
				return err
			}
		}
	}
	return nil
}

/* -------------------------------------------------------------------------- */

type BbiZoomBlockDecoder struct {
	Buffer []byte
}

type BbiZoomBlockDecoderType struct {
	BbiSummaryRecord
	Error error
}

func NewBbiZoomBlockDecoder(buffer []byte) *BbiZoomBlockDecoder {
	return &BbiZoomBlockDecoder{buffer}
}

func (reader *BbiZoomBlockDecoder) decode(channel chan BbiZoomBlockDecoderType) {
	b := bytes.NewReader(reader.Buffer)
	for {
		record := BbiZoomRecord{}
		if err := record.Read(b); err != nil {
			break
		} else {
			result := BbiZoomBlockDecoderType{}
			result.ChromId = int(record.ChromId)
			result.From = int(record.Start)
			result.To = int(record.End)
			result.Valid = int(record.Valid)
			result.Min = float64(record.Min)
			result.Max = float64(record.Max)
			result.Sum = float64(record.Sum)
			result.SumSquares = float64(record.SumSquares)
			channel <- result
		}
	}
}

func (reader *BbiZoomBlockDecoder) Decode() <-chan BbiZoomBlockDecoderType {
	channel := make(chan BbiZoomBlockDecoderType)
	go func() {
		reader.decode(channel)
		close(channel)
	}()
	return channel
}

/* -------------------------------------------------------------------------- */

type BbiBlockEncoder struct {
	ItemsPerSlot int
	tmp          []byte
}

type BbiBlockEncoderType struct {
	From  int
	To    int
	Block []byte
	Error error
}

func NewBbiBlockEncoder(itemsPerSlot int) (*BbiBlockEncoder, error) {
	writer := BbiBlockEncoder{}
	writer.ItemsPerSlot = itemsPerSlot
	writer.tmp = make([]byte, 8)
	return &writer, nil
}

func (writer *BbiBlockEncoder) encodeBlockZoom(channel chan BbiBlockEncoderType, chromid int, sequence []float64, binsize, reductionLevel int) {
	b := new(bytes.Buffer)
	w := bufio.NewWriter(b)
	// number of bins covered by each record
	n := divIntUp(reductionLevel, binsize)
	// beginning of region covered by a single block
	f := -1
	// number of records written to block
	m := 0
	// loop over sequence and generate records
	for p := 0; p < binsize*len(sequence); p += reductionLevel {
		// p: position in base pairs
		// i: position in bins
		i := p / binsize
		// create a new record
		record := BbiZoomRecord{
			ChromId: uint32(chromid),
			Start:   uint32(p),
			End:     uint32(p + reductionLevel),
			Min:     math.MaxFloat32,
			Max:     -math.MaxFloat32}
		// crop record end if it is longer than the actual sequence
		if record.End > uint32(binsize*len(sequence)) {
			record.End = uint32(binsize * len(sequence))
		}
		// compute mean value
		for j := 0; j < n && i+j < len(sequence); j++ {
			if record.Min > float32(sequence[i+j]) {
				record.Min = float32(sequence[i+j])
			}
			if record.Max < float32(sequence[i+j]) {
				record.Max = float32(sequence[i+j])
			}
			record.Valid += 1
			record.Sum += float32(sequence[i+j])
			record.SumSquares += float32(sequence[i+j] * sequence[i+j])
		}
		// check if there was some non-zero data
		if !(record.Min == 0 && record.Max == 0) {
			// if yes, save record
			if err := record.Write(w); err != nil {
				channel <- BbiBlockEncoderType{Error: err}
				return
			}
			// if this is the first record in a block
			if f == -1 {
				// save position
				f = int(record.Start)
			}
			m += 1
		}
		// check if block is full or if the end of the
		// sequence is reached
		if m == writer.ItemsPerSlot || p+reductionLevel >= binsize*len(sequence) {
			w.Flush()
			if tmp := b.Bytes(); len(tmp) > 0 {
				if f == -1 {
					// this shouldn't happen
					// panic("internal error")
				}
				// send block over channel
				channel <- BbiBlockEncoderType{
					From:  f,
					To:    int(record.End),
					Block: tmp}
				// reset buffer
				b = new(bytes.Buffer)
				w = bufio.NewWriter(b)
				// reset variables
				m = 0
				f = -1
			}
		}
	}
}

func (writer *BbiBlockEncoder) encodeVariable(buffer []byte, position uint32, value float64) {
	binary.LittleEndian.PutUint32(buffer[0:4], position)
	binary.LittleEndian.PutUint32(buffer[4:8], math.Float32bits(float32(value)))
}

func (writer *BbiBlockEncoder) encodeFixed(buffer []byte, value float64) {
	binary.LittleEndian.PutUint32(buffer[0:4], math.Float32bits(float32(value)))
}

func (writer *BbiBlockEncoder) encodeBlock(channel chan BbiBlockEncoderType, chromid int, sequence []float64, binsize int, fixedStep bool) {
	// loop over sequence and generate records
	for i := 0; i < len(sequence); {
		// create header for this block
		header := BbiDataHeader{}
		header.ChromId = uint32(chromid)
		header.Start = uint32(i * binsize)
		header.End = uint32(i * binsize)
		header.Step = uint32(binsize)
		header.Span = uint32(binsize)
		if fixedStep {
			header.Type = 3
		} else {
			header.Type = 2
		}
		// new buffer
		var buffer bytes.Buffer
		// fill buffer with data
		switch header.Type {
		default:
			channel <- BbiBlockEncoderType{
				Error: fmt.Errorf("unsupported block type")}
			return
		case 2:
			// variable step
			for ; i < len(sequence); i++ {
				if sequence[i] != 0.0 && !math.IsNaN(sequence[i]) {
					writer.encodeVariable(writer.tmp, header.End, sequence[i])
					if _, err := buffer.Write(writer.tmp); err != nil {
						channel <- BbiBlockEncoderType{Error: err}
						return
					}
					header.ItemCount++
				}
				header.End += header.Step
				// check if maximum number of items per block is reached
				if int(header.ItemCount) == writer.ItemsPerSlot {
					break
				}
			}
		case 3:
			// fixed step
			for ; i < len(sequence); i++ {
				writer.encodeFixed(writer.tmp, sequence[i])
				if _, err := buffer.Write(writer.tmp[0:4]); err != nil {
					channel <- BbiBlockEncoderType{Error: err}
					return
				}
				header.ItemCount++
				header.End += header.Step
				// check if maximum number of items per block is reached
				if int(header.ItemCount) == writer.ItemsPerSlot {
					break
				}
			}
		}
		if tmp := buffer.Bytes(); len(tmp) > 0 {
			// prepend header
			block := make([]byte, 24)
			header.WriteBuffer(block)
			block = append(block, tmp...)
			// send result through channel
			channel <- BbiBlockEncoderType{
				From:  int(header.Start),
				To:    int(header.End),
				Block: block}
		}
	}
}

func (writer *BbiBlockEncoder) EncodeBlock(chromid int, sequence []float64, binsize, reductionLevel int, fixedStep bool) <-chan BbiBlockEncoderType {
	// open a new channel
	channel := make(chan BbiBlockEncoderType)
	// fill channel
	go func() {
		if reductionLevel > binsize {
			writer.encodeBlockZoom(channel, chromid, sequence, binsize, reductionLevel)
		} else {
			writer.encodeBlock(channel, chromid, sequence, binsize, fixedStep)
		}
		close(channel)
	}()
	return channel
}

/* -------------------------------------------------------------------------- */

type BTree struct {
	KeySize       uint32
	ValueSize     uint32
	ItemsPerBlock uint32
	ItemCount     uint64
	Root          BVertex
}

type BVertex struct {
	IsLeaf   uint8
	Keys     [][]byte
	Values   [][]byte
	Children []BVertex
}

func NewBTree(data *BData) *BTree {
	tree := BTree{}
	tree.KeySize = data.KeySize
	tree.ValueSize = data.ValueSize
	tree.ItemsPerBlock = data.ItemsPerBlock
	tree.ItemCount = data.ItemCount
	// compute tree depth
	d := int(math.Ceil(math.Log(float64(data.ItemCount)) / math.Log(float64(data.ItemsPerBlock))))

	tree.Root.BuildTree(data, 0, data.ItemCount, d-1)

	return &tree
}

func (vertex *BVertex) BuildTree(data *BData, from, to uint64, level int) (uint64, error) {
	// number of values below this node
	i := uint64(0)
	if level == 0 {
		vertex.IsLeaf = 1
		for nVals := uint16(0); uint32(nVals) < data.ItemsPerBlock && from+i < to; nVals++ {
			if uint32(len(data.Keys[from+i])) != data.KeySize {
				return 0, fmt.Errorf("key number `%d' has invalid size", i)
			}
			if uint32(len(data.Values[from+i])) != data.ValueSize {
				return 0, fmt.Errorf("value number `%d' has invalid size", i)
			}
			vertex.Keys = append(vertex.Keys, data.Keys[from+i])
			vertex.Values = append(vertex.Values, data.Values[from+i])
			i++
		}
	} else {
		vertex.IsLeaf = 0
		for nVals := uint16(0); uint32(nVals) < data.ItemsPerBlock && from+i < to; nVals++ {
			// append first key
			vertex.Keys = append(vertex.Keys, data.Keys[from+i])
			// create new child vertex
			v := BVertex{}
			if j, err := v.BuildTree(data, from+i, to, level-1); err != nil {
				return 0, err
			} else {
				i += j
			}
			// append child
			vertex.Children = append(vertex.Children, v)
		}
	}
	return i, nil
}

func (vertex *BVertex) writeLeaf(file io.Writer) error {
	padding := uint8(0)
	nVals := uint16(len(vertex.Keys))

	if err := binary.Write(file, binary.LittleEndian, vertex.IsLeaf); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, padding); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, nVals); err != nil {
		return err
	}
	for i := 0; i < len(vertex.Keys); i++ {
		if err := binary.Write(file, binary.LittleEndian, vertex.Keys[i]); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, vertex.Values[i]); err != nil {
			return err
		}
	}
	return nil
}

func (vertex *BVertex) writeIndex(file io.WriteSeeker) error {
	isLeaf := uint8(0)
	padding := uint8(0)
	nVals := uint16(len(vertex.Keys))
	offsets := make([]int64, nVals)

	if err := binary.Write(file, binary.LittleEndian, isLeaf); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, padding); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, nVals); err != nil {
		return err
	}
	for i := 0; i < int(nVals); i++ {
		if err := binary.Write(file, binary.LittleEndian, vertex.Keys[i]); err != nil {
			return err
		}
		// save current file offset
		offsets[i], _ = file.Seek(0, 1)
		// offset of the ith child vertex (first set to zero)
		if err := binary.Write(file, binary.LittleEndian, uint64(0)); err != nil {
			return err
		}
	}
	// write child vertices
	for i := 0; i < int(nVals); i++ {
		// get current file offset (where the ith child vertex begins)
		offset, _ := file.Seek(0, 1)
		// and write it at the expected position
		if err := fileWriteAt(file, binary.LittleEndian, offsets[i], uint64(offset)); err != nil {
			return err
		}
		// write ith child
		if err := vertex.Children[i].write(file); err != nil {
			return err
		}
	}
	return nil
}

func (vertex *BVertex) write(file io.WriteSeeker) error {
	if vertex.IsLeaf != 0 {
		return vertex.writeLeaf(file)
	} else {
		return vertex.writeIndex(file)
	}
	return nil
}

func (tree *BTree) Write(file io.WriteSeeker) error {
	magic := uint32(CIRTREE_MAGIC)

	// ItemsPerBlock has 32 bits but nVals has only 16 bits, check for overflow
	if tree.ItemsPerBlock > uint32(^uint16(0)) {
		return fmt.Errorf("ItemsPerBlock too large (maximum value is `%d')", ^uint16(0))
	}

	if err := binary.Write(file, binary.LittleEndian, magic); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.ItemsPerBlock); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.KeySize); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.ValueSize); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.ItemCount); err != nil {
		return err
	}
	// padding
	if err := binary.Write(file, binary.LittleEndian, uint64(0)); err != nil {
		return err
	}
	return tree.Root.write(file)
}

/* -------------------------------------------------------------------------- */

type BData struct {
	KeySize       uint32
	ValueSize     uint32
	ItemsPerBlock uint32
	ItemCount     uint64

	Keys   [][]byte
	Values [][]byte
}

func NewBData() *BData {
	data := BData{}
	// default values
	data.KeySize = 0
	data.ValueSize = 0
	data.ItemsPerBlock = 0
	data.ItemCount = 0
	return &data
}

func (data *BData) Add(key, value []byte) error {
	if uint32(len(key)) != data.KeySize {
		return fmt.Errorf("BData.Add(): key has invalid length")
	}
	if uint32(len(value)) != data.ValueSize {
		return fmt.Errorf("BData.Add(): value has invalid length")
	}
	data.Keys = append(data.Keys, key)
	data.Values = append(data.Values, value)
	data.ItemsPerBlock++
	data.ItemCount++
	return nil
}

func (data *BData) readVertexLeaf(file io.ReadSeeker) error {
	var nVals uint16
	var key []byte
	var value []byte

	if err := binary.Read(file, binary.LittleEndian, &nVals); err != nil {
		return err
	}

	for i := 0; i < int(nVals); i++ {
		key = make([]byte, data.KeySize)
		value = make([]byte, data.ValueSize)
		if err := binary.Read(file, binary.LittleEndian, &key); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &value); err != nil {
			return err
		}
		data.Keys = append(data.Keys, key)
		data.Values = append(data.Values, value)
	}
	return nil
}

func (data *BData) readVertexIndex(file io.ReadSeeker) error {
	var nVals uint16
	var key []byte
	var position uint64

	key = make([]byte, data.KeySize)

	if err := binary.Read(file, binary.LittleEndian, &nVals); err != nil {
		return err
	}

	for i := 0; i < int(nVals); i++ {
		if err := binary.Read(file, binary.LittleEndian, &key); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &position); err != nil {
			return err
		}
		// save current position and jump to child vertex
		currentPosition, _ := file.Seek(0, 1)
		if _, err := file.Seek(int64(position), 0); err != nil {
			return err
		}
		data.readVertex(file)
		if _, err := file.Seek(currentPosition, 0); err != nil {
			return err
		}
	}
	return nil
}

func (data *BData) readVertex(file io.ReadSeeker) error {
	var isLeaf uint8
	var padding uint8

	if err := binary.Read(file, binary.LittleEndian, &isLeaf); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &padding); err != nil {
		return err
	}
	if isLeaf != 0 {
		return data.readVertexLeaf(file)
	} else {
		return data.readVertexIndex(file)
	}
}

func (data *BData) Read(file io.ReadSeeker) error {

	var magic uint32

	// magic number
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return err
	}
	if magic != CIRTREE_MAGIC {
		return fmt.Errorf("invalid tree")
	}

	if err := binary.Read(file, binary.LittleEndian, &data.ItemsPerBlock); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &data.KeySize); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &data.ValueSize); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &data.ItemCount); err != nil {
		return err
	}
	// padding
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return err
	}
	return data.readVertex(file)
}

func (data *BData) Write(file io.WriteSeeker) error {
	tree := NewBTree(data)
	return tree.Write(file)
}

/* -------------------------------------------------------------------------- */

type BbiDataHeader struct {
	ChromId   uint32
	Start     uint32
	End       uint32
	Step      uint32
	Span      uint32
	Type      byte
	Reserved  byte
	ItemCount uint16
}

func (header *BbiDataHeader) ReadBuffer(buffer []byte) {

	header.ChromId = binary.LittleEndian.Uint32(buffer[0:4])
	header.Start = binary.LittleEndian.Uint32(buffer[4:8])
	header.End = binary.LittleEndian.Uint32(buffer[8:12])
	header.Step = binary.LittleEndian.Uint32(buffer[12:16])
	header.Span = binary.LittleEndian.Uint32(buffer[16:20])
	header.Type = buffer[20]
	header.Reserved = buffer[21]
	header.ItemCount = binary.LittleEndian.Uint16(buffer[22:24])

}

func (header *BbiDataHeader) WriteBuffer(buffer []byte) {

	binary.LittleEndian.PutUint32(buffer[0:4], header.ChromId)
	binary.LittleEndian.PutUint32(buffer[4:8], header.Start)
	binary.LittleEndian.PutUint32(buffer[8:12], header.End)
	binary.LittleEndian.PutUint32(buffer[12:16], header.Step)
	binary.LittleEndian.PutUint32(buffer[16:20], header.Span)
	buffer[20] = header.Type
	buffer[21] = header.Reserved
	binary.LittleEndian.PutUint16(buffer[22:24], header.ItemCount)

}

/* -------------------------------------------------------------------------- */

type RTree struct {
	BlockSize     uint32
	NItems        uint64
	ChrIdxStart   uint32
	BaseStart     uint32
	ChrIdxEnd     uint32
	BaseEnd       uint32
	IdxSize       uint64
	NItemsPerSlot uint32
	Root          *RVertex
	PtrIdxSize    int64
}

func NewRTree() *RTree {
	tree := RTree{}
	// default values
	tree.BlockSize = 256
	tree.NItemsPerSlot = 1024
	return &tree
}

func (tree *RTree) Read(file io.ReadSeeker) error {

	var magic uint32

	// magic number
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return err
	}
	if magic != IDX_MAGIC {
		return fmt.Errorf("invalid bbi tree")
	}

	if err := binary.Read(file, binary.LittleEndian, &tree.BlockSize); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.NItems); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.ChrIdxStart); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.BaseStart); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.ChrIdxEnd); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.BaseEnd); err != nil {
		return err
	}
	// get current offset
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		tree.PtrIdxSize = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.IdxSize); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &tree.NItemsPerSlot); err != nil {
		return err
	}
	// padding
	if err := binary.Read(file, binary.LittleEndian, &magic); err != nil {
		return err
	}
	tree.Root = new(RVertex)
	tree.Root.Read(file)

	return nil
}

func (tree *RTree) WriteSize(file io.WriteSeeker) error {
	return fileWriteAt(file, binary.LittleEndian, tree.PtrIdxSize, tree.IdxSize)
}

func (tree *RTree) Write(file io.WriteSeeker) error {
	var offsetStart int64
	var offsetEnd int64
	// get current offset
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		offsetStart = offset
	}
	// magic number
	if err := binary.Write(file, binary.LittleEndian, uint32(IDX_MAGIC)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.BlockSize); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.NItems); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.ChrIdxStart); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.BaseStart); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.ChrIdxEnd); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.BaseEnd); err != nil {
		return err
	}
	// get current offset
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		tree.PtrIdxSize = offset
	}
	if err := binary.Write(file, binary.LittleEndian, tree.IdxSize); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, tree.NItemsPerSlot); err != nil {
		return err
	}
	// padding
	if err := binary.Write(file, binary.LittleEndian, uint32(0)); err != nil {
		return err
	}
	tree.Root.Write(file)
	// get current offset
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		offsetEnd = offset
	}
	// update index size
	tree.IdxSize = uint64(offsetEnd - offsetStart)
	tree.WriteSize(file)

	return nil
}

func (tree *RTree) buildTreeRec(leaves []*RVertex, level int) (*RVertex, []*RVertex) {
	v := new(RVertex)
	n := len(leaves)
	// return if there are no leaves
	if n == 0 {
		return nil, leaves
	}
	if level == 0 {
		if n > int(tree.BlockSize) {
			n = int(tree.BlockSize)
		}
		v.NChildren = uint16(n)
		v.Children = leaves[0:n]
		// update free leaf set
		leaves = leaves[n:]
	} else {
		for i := 0; i < int(tree.BlockSize) && len(leaves) > 0; i++ {
			var vertex *RVertex
			vertex, leaves = tree.buildTreeRec(leaves, level-1)
			v.NChildren++
			v.Children = append(v.Children, vertex)
		}
	}
	for i := 0; i < len(v.Children); i++ {
		v.ChrIdxStart = append(v.ChrIdxStart, v.Children[i].ChrIdxStart[0])
		v.ChrIdxEnd = append(v.ChrIdxEnd, v.Children[i].ChrIdxEnd[v.Children[i].NChildren-1])
		v.BaseStart = append(v.BaseStart, v.Children[i].BaseStart[0])
		v.BaseEnd = append(v.BaseEnd, v.Children[i].BaseEnd[v.Children[i].NChildren-1])
	}
	return v, leaves
}

func (tree *RTree) BuildTree(leaves []*RVertex) error {
	if len(leaves) == 0 {
		return nil
	}
	if len(leaves) == 1 {
		tree.Root = leaves[0]
	} else {
		// compute tree depth
		d := int(math.Ceil(math.Log(float64(len(leaves))) / math.Log(float64(tree.BlockSize))))
		// construct tree
		if root, leaves := tree.buildTreeRec(leaves, d-1); len(leaves) != 0 {
			// panic("internal error")
			return errors.New("internal error")
		} else {
			tree.Root = root
		}
	}
	tree.ChrIdxStart = tree.Root.ChrIdxStart[0]
	tree.ChrIdxEnd = tree.Root.ChrIdxEnd[tree.Root.NChildren-1]
	tree.BaseStart = tree.Root.BaseStart[0]
	tree.BaseEnd = tree.Root.BaseEnd[tree.Root.NChildren-1]
	return nil
}

/* -------------------------------------------------------------------------- */

type RTreeTraverser struct {
	Tree *RTree
}

type RTreeTraverserType struct {
	Vertex *RVertex
	Idx    int
}

func NewRTreeTraverser(tree *RTree) RTreeTraverser {
	return RTreeTraverser{tree}
}

func (traverser RTreeTraverser) queryVertices(channel chan RTreeTraverserType, vertex *RVertex, idx, from, to int) {
	if vertex == nil {
		return
	}
	for i := 0; i < int(vertex.NChildren); i++ {
		// check if this is the correct chromosome
		if idx >= int(vertex.ChrIdxStart[i]) && idx <= int(vertex.ChrIdxEnd[i]) {
			if int(vertex.ChrIdxStart[i]) == int(vertex.ChrIdxEnd[i]) {
				// check region on chromosome
				if int(vertex.BaseEnd[i]) <= from {
					// query region is still ahead
					continue
				}
				if int(vertex.BaseStart[i]) >= to {
					// already past the query region
					break
				}
			}
			// found a match
			if vertex.IsLeaf == 0 {
				traverser.queryVertices(channel, vertex.Children[i], idx, from, to)
			} else {
				channel <- RTreeTraverserType{vertex, i}
			}
		}
		// indices are sorted, hence stop searching if idx is larger than the
		// curent index end
		if int(vertex.ChrIdxStart[i]) > idx {
			break
		}
	}
}

func (traverser RTreeTraverser) QueryVertices(idx, from, to int) <-chan RTreeTraverserType {
	channel := make(chan RTreeTraverserType)
	go func() {
		traverser.queryVertices(channel, traverser.Tree.Root, idx, from, to)
		close(channel)
	}()
	return channel
}

/* -------------------------------------------------------------------------- */

type RVertex struct {
	IsLeaf      uint8
	NChildren   uint16
	ChrIdxStart []uint32
	BaseStart   []uint32
	ChrIdxEnd   []uint32
	BaseEnd     []uint32
	DataOffset  []uint64
	Sizes       []uint64
	Children    []*RVertex
	// positions of DataOffset and Sizes values in file
	PtrDataOffset []int64
	PtrSizes      []int64
}

/* TODO MV BbiFile to BbiWriteSeeker
func (vertex *RVertex) WriteBlock(bwf *BbiWriteSeeker, i int, block []byte) error {
	var err error
	if bwf.Header.UncompressBufSize != 0 {
		// update header.UncompressBufSize if block length
		// exceeds size
		if uint32(len(block)) > bwf.Header.UncompressBufSize {
			bwf.Header.UncompressBufSize = uint32(len(block))
			if err = bwf.Header.WriteUncompressBufSize(bwf.Fptr); err != nil {
				return err
			}
		}
		if block, err = compressSlice(block); err != nil {
			return err
		}
	}
	// get current offset and update DataOffset[i]
	if offset, err := bwf.Fptr.Seek(0, 1); err != nil {
		return err
	} else {
		vertex.DataOffset[i] = uint64(offset)
		// write updated value to the required position in the file
		if vertex.PtrDataOffset[i] != 0 {
			if err = fileWriteAt(bwf.Fptr, binary.LittleEndian, int64(vertex.PtrDataOffset[i]), vertex.DataOffset[i]); err != nil {
				return err
			}
		}
	}
	// write data
	if err = binary.Write(bwf.Fptr, binary.LittleEndian, block); err != nil {
		return err
	}
	// update size of the data block
	vertex.Sizes[i] = uint64(len(block))
	// write it to the required position in the file
	if vertex.PtrSizes[i] != 0 {
		if err = fileWriteAt(bwf.Fptr, binary.LittleEndian, int64(vertex.PtrSizes[i]), vertex.Sizes[i]); err != nil {
			return err
		}
	}
	return nil
}
*/
func (vertex *RVertex) Read(file io.ReadSeeker) error {

	var padding uint8

	if err := binary.Read(file, binary.LittleEndian, &vertex.IsLeaf); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &padding); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &vertex.NChildren); err != nil {
		return err
	}
	// allocate data
	vertex.ChrIdxStart = make([]uint32, vertex.NChildren)
	vertex.BaseStart = make([]uint32, vertex.NChildren)
	vertex.ChrIdxEnd = make([]uint32, vertex.NChildren)
	vertex.BaseEnd = make([]uint32, vertex.NChildren)
	vertex.DataOffset = make([]uint64, vertex.NChildren)
	vertex.PtrDataOffset = make([]int64, vertex.NChildren)
	if vertex.IsLeaf != 0 {
		vertex.Sizes = make([]uint64, vertex.NChildren)
		vertex.PtrSizes = make([]int64, vertex.NChildren)
	} else {
		vertex.Children = make([]*RVertex, vertex.NChildren)
	}

	for i := 0; i < int(vertex.NChildren); i++ {
		if err := binary.Read(file, binary.LittleEndian, &vertex.ChrIdxStart[i]); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &vertex.BaseStart[i]); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &vertex.ChrIdxEnd[i]); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &vertex.BaseEnd[i]); err != nil {
			return err
		}
		if offset, err := file.Seek(0, 1); err != nil {
			return err
		} else {
			vertex.PtrDataOffset[i] = offset
		}
		if err := binary.Read(file, binary.LittleEndian, &vertex.DataOffset[i]); err != nil {
			return err
		}
		if vertex.IsLeaf != 0 {
			if offset, err := file.Seek(0, 1); err != nil {
				return err
			} else {
				vertex.PtrSizes[i] = offset
			}
			if err := binary.Read(file, binary.LittleEndian, &vertex.Sizes[i]); err != nil {
				return err
			}
		}
	}
	if vertex.IsLeaf == 0 {
		for i := 0; i < int(vertex.NChildren); i++ {
			// seek to child position
			if _, err := file.Seek(int64(vertex.DataOffset[i]), 0); err != nil {
				return err
			}
			vertex.Children[i] = new(RVertex)
			vertex.Children[i].Read(file)
		}
	}
	return nil
}

func (vertex *RVertex) Write(file io.WriteSeeker) error {

	if len(vertex.DataOffset) != int(vertex.NChildren) {
		vertex.DataOffset = make([]uint64, vertex.NChildren)
	}
	if len(vertex.Sizes) != int(vertex.NChildren) {
		vertex.Sizes = make([]uint64, vertex.NChildren)
	}
	if len(vertex.PtrDataOffset) != int(vertex.NChildren) {
		vertex.PtrDataOffset = make([]int64, vertex.NChildren)
	}
	if len(vertex.PtrSizes) != int(vertex.NChildren) {
		vertex.PtrSizes = make([]int64, vertex.NChildren)
	}

	if err := binary.Write(file, binary.LittleEndian, vertex.IsLeaf); err != nil {
		return err
	}
	// padding
	if err := binary.Write(file, binary.LittleEndian, uint8(0)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, vertex.NChildren); err != nil {
		return err
	}

	for i := 0; i < int(vertex.NChildren); i++ {
		if err := binary.Write(file, binary.LittleEndian, vertex.ChrIdxStart[i]); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, vertex.BaseStart[i]); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, vertex.ChrIdxEnd[i]); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, vertex.BaseEnd[i]); err != nil {
			return err
		}
		// save current offset
		if offset, err := file.Seek(0, 1); err != nil {
			return err
		} else {
			vertex.PtrDataOffset[i] = offset
		}
		if err := binary.Write(file, binary.LittleEndian, vertex.DataOffset[i]); err != nil {
			return err
		}
		// save current offset
		if offset, err := file.Seek(0, 1); err != nil {
			return err
		} else {
			vertex.PtrSizes[i] = offset
		}
		if vertex.IsLeaf != 0 {
			if err := binary.Write(file, binary.LittleEndian, vertex.Sizes[i]); err != nil {
				return err
			}
		}
	}
	if vertex.IsLeaf == 0 {
		for i := 0; i < int(vertex.NChildren); i++ {
			if offset, err := file.Seek(0, 1); err != nil {
				return err
			} else {
				// save current offset
				vertex.DataOffset[i] = uint64(offset)
				// and write at the required position
				fileWriteAt(file, binary.LittleEndian, vertex.PtrDataOffset[i], vertex.DataOffset[i])
				vertex.Children[i].Write(file)
			}
		}
	}
	return nil
}

/* -------------------------------------------------------------------------- */

type RVertexGenerator struct {
	BlockSize    int
	ItemsPerSlot int
	Channel      chan RVertexGeneratorType
}

type RVertexGeneratorType struct {
	Vertex *RVertex
	Blocks [][]byte
}

func NewRVertexGenerator(blockSize, itemsPerSlot int) (*RVertexGenerator, error) {
	if blockSize <= 0 {
		return nil, fmt.Errorf("invalid block size `%d'", blockSize)
	}
	if itemsPerSlot <= 0 {
		return nil, fmt.Errorf("invalid items per slot `%d'", itemsPerSlot)
	}
	generator := RVertexGenerator{}
	generator.BlockSize = blockSize
	generator.ItemsPerSlot = itemsPerSlot
	return &generator, nil
}

func (generator *RVertexGenerator) Generate(idx int, sequence []float64, binsize, reductionLevel int, fixedStep bool) <-chan RVertexGeneratorType {
	generator.Channel = make(chan RVertexGeneratorType)
	go func() {
		generator.fillChannel(idx, sequence, binsize, reductionLevel, fixedStep)
		close(generator.Channel)
	}()
	return generator.Channel
}

func (generator *RVertexGenerator) fillChannel(idx int, sequence []float64, binsize, reductionLevel int, fixedStep bool) error {
	var encoder *BbiBlockEncoder
	// create block encoder
	if tmp, err := NewBbiBlockEncoder(generator.ItemsPerSlot); err != nil {
		return err
	} else {
		encoder = tmp
	}
	// create empty leaf
	v := new(RVertex)
	v.IsLeaf = 1
	// empty list of blocks
	b := [][]byte{}
	// loop over sequence chunks
	for chunk := range encoder.EncodeBlock(idx, sequence, binsize, reductionLevel, fixedStep) {
		if chunk.Error != nil {
			return chunk.Error
		}
		if int(v.NChildren) == generator.BlockSize {
			// vertex is full
			generator.Channel <- RVertexGeneratorType{
				Vertex: v,
				Blocks: b}
			// create new emtpy vertex
			v = new(RVertex)
			v.IsLeaf = 1
			b = [][]byte{}
		}
		v.ChrIdxStart = append(v.ChrIdxStart, uint32(idx))
		v.ChrIdxEnd = append(v.ChrIdxEnd, uint32(idx))
		v.BaseStart = append(v.BaseStart, uint32(chunk.From))
		v.BaseEnd = append(v.BaseEnd, uint32(chunk.To))
		v.DataOffset = append(v.DataOffset, 0)
		v.Sizes = append(v.Sizes, 0)
		v.PtrDataOffset = append(v.PtrDataOffset, 0)
		v.PtrSizes = append(v.PtrSizes, 0)
		v.NChildren++
		b = append(b, chunk.Block)
	}
	if v.NChildren != 0 {
		generator.Channel <- RVertexGeneratorType{
			Vertex: v,
			Blocks: b}
	}
	return nil
}

/* -------------------------------------------------------------------------- */

type BbiHeaderZoom struct {
	ReductionLevel uint32
	Reserved       uint32
	DataOffset     uint64
	IndexOffset    uint64
	NBlocks        uint32
	PtrDataOffset  int64
	PtrIndexOffset int64
}

func (zoomHeader *BbiHeaderZoom) Read(file io.ReadSeeker) error {

	if err := binary.Read(file, binary.LittleEndian, &zoomHeader.ReductionLevel); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &zoomHeader.Reserved); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		zoomHeader.PtrDataOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &zoomHeader.DataOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		zoomHeader.PtrIndexOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &zoomHeader.IndexOffset); err != nil {
		return err
	}
	return nil
}

func (zoomHeader *BbiHeaderZoom) Write(file io.WriteSeeker) error {

	if err := binary.Write(file, binary.LittleEndian, zoomHeader.ReductionLevel); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, zoomHeader.Reserved); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		zoomHeader.PtrDataOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, zoomHeader.DataOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		zoomHeader.PtrIndexOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, zoomHeader.IndexOffset); err != nil {
		return err
	}
	return nil
}

func (zoomHeader *BbiHeaderZoom) WriteOffsets(file io.WriteSeeker) error {
	if zoomHeader.PtrDataOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, zoomHeader.PtrDataOffset, zoomHeader.DataOffset); err != nil {
			return err
		}
	}
	if zoomHeader.PtrIndexOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, zoomHeader.PtrIndexOffset, zoomHeader.IndexOffset); err != nil {
			return err
		}
	}
	return nil
}

func (zoomHeader *BbiHeaderZoom) WriteNBlocks(file io.WriteSeeker) error {
	return fileWriteAt(file, binary.LittleEndian, int64(zoomHeader.DataOffset), zoomHeader.NBlocks)
}

/* -------------------------------------------------------------------------- */

type BbiHeader struct {
	Magic             uint32
	Version           uint16
	ZoomLevels        uint16
	CtOffset          uint64
	DataOffset        uint64
	IndexOffset       uint64
	FieldCould        uint16
	DefinedFieldCount uint16
	SqlOffset         uint64
	SummaryOffset     uint64
	UncompressBufSize uint32
	ExtensionOffset   uint64
	NBasesCovered     uint64
	MinVal            uint64
	MaxVal            uint64
	SumData           uint64
	SumSquared        uint64
	ZoomHeaders       []BbiHeaderZoom
	NBlocks           uint64
	// offset positions
	PtrCtOffset          int64
	PtrDataOffset        int64
	PtrIndexOffset       int64
	PtrSqlOffset         int64
	PtrSummaryOffset     int64
	PtrUncompressBufSize int64
	PtrExtensionOffset   int64
}

func NewBbiHeader() *BbiHeader {
	header := BbiHeader{}
	header.Version = 4
	return &header
}

//func (header *BbiHeader) Read(file *os.File) error {
func (header *BbiHeader) Read(file io.ReadSeeker) error {
	if err := binary.Read(file, binary.LittleEndian, &header.Magic); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &header.Version); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &header.ZoomLevels); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrCtOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.CtOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrDataOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.DataOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrIndexOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.IndexOffset); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &header.FieldCould); err != nil {
		return err
	}
	if err := binary.Read(file, binary.LittleEndian, &header.DefinedFieldCount); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrSqlOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.SqlOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrSummaryOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.SummaryOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrUncompressBufSize = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.UncompressBufSize); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrExtensionOffset = offset
	}
	if err := binary.Read(file, binary.LittleEndian, &header.ExtensionOffset); err != nil {
		return err
	}
	// zoom levels
	header.ZoomHeaders = make([]BbiHeaderZoom, header.ZoomLevels)
	for i := 0; i < int(header.ZoomLevels); i++ {
		if err := header.ZoomHeaders[i].Read(file); err != nil {
			return err
		}
	}
	// summary
	if header.SummaryOffset > 0 {
		if _, err := file.Seek(int64(header.SummaryOffset), 0); err != nil {
			return err
		}

		if err := binary.Read(file, binary.LittleEndian, &header.NBasesCovered); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &header.MinVal); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &header.MaxVal); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &header.SumData); err != nil {
			return err
		}
		if err := binary.Read(file, binary.LittleEndian, &header.SumSquared); err != nil {
			return err
		}
	}
	return nil
}

func (header *BbiHeader) WriteOffsets(file io.WriteSeeker) error {
	if header.PtrCtOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, header.PtrCtOffset, header.CtOffset); err != nil {
			return err
		}
	}
	if header.PtrDataOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, header.PtrDataOffset, header.DataOffset); err != nil {
			return err
		}
	}
	if header.PtrIndexOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, header.PtrIndexOffset, header.IndexOffset); err != nil {
			return err
		}
	}
	if header.PtrSqlOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, header.PtrSqlOffset, header.SqlOffset); err != nil {
			return err
		}
	}
	if header.PtrExtensionOffset != 0 {
		if err := fileWriteAt(file, binary.LittleEndian, header.PtrExtensionOffset, header.ExtensionOffset); err != nil {
			return err
		}
	}
	return nil
}

func (header *BbiHeader) WriteUncompressBufSize(file io.WriteSeeker) error {
	if header.PtrUncompressBufSize != 0 {
		return fileWriteAt(file, binary.LittleEndian, header.PtrUncompressBufSize, header.UncompressBufSize)
	}
	return nil
}

func (header *BbiHeader) Write(file io.WriteSeeker) error {

	if err := binary.Write(file, binary.LittleEndian, header.Magic); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, header.Version); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, header.ZoomLevels); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrCtOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.CtOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrDataOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.DataOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrIndexOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.IndexOffset); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, header.FieldCould); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, header.DefinedFieldCount); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrSqlOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.SqlOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrSummaryOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.SummaryOffset); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrUncompressBufSize = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.UncompressBufSize); err != nil {
		return err
	}
	if offset, err := file.Seek(0, 1); err != nil {
		return err
	} else {
		header.PtrExtensionOffset = offset
	}
	if err := binary.Write(file, binary.LittleEndian, header.ExtensionOffset); err != nil {
		return err
	}
	// zoom levels
	for i := 0; i < int(header.ZoomLevels); i++ {
		if err := header.ZoomHeaders[i].Write(file); err != nil {
			return err
		}
	}
	// summary
	if header.NBasesCovered > 0 {
		// get current offset
		if offset, err := file.Seek(0, 1); err != nil {
			return err
		} else {
			header.SummaryOffset = uint64(offset)
			// write curent offset to the position of SummaryOffset
			if err := fileWriteAt(file, binary.LittleEndian, header.PtrSummaryOffset, header.SummaryOffset); err != nil {
				return err
			}
		}
		if err := binary.Write(file, binary.LittleEndian, header.NBasesCovered); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, header.MinVal); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, header.MaxVal); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, header.SumData); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, header.SumSquared); err != nil {
			return err
		}
	}

	return nil
}

func (header *BbiHeader) WriteNBlocks(file io.WriteSeeker) error {
	return fileWriteAt(file, binary.LittleEndian, int64(header.DataOffset), header.NBlocks)
}

/* -------------------------------------------------------------------------- */

type BbiQueryType struct {
	BbiSummaryRecord
	Error error
}

func NewBbiQueryType() *BbiQueryType {
	return &BbiQueryType{NewBbiSummaryRecord(), nil}
}
