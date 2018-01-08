package hic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/nimezhu/indexed/hic/normtype"
	"github.com/nimezhu/indexed/hic/unit"
	. "github.com/nimezhu/netio"
)

const (
	maxBufferSize = 1000
)

type HiC struct {
	Reader            io.ReadSeeker
	mutex             *sync.Mutex
	Version           int32
	masterIndexPos    int64
	Genome            string
	NAttr             int32
	Attr              map[string]string
	NChrs             int32
	Chr               []Chr
	NBpRes            int32
	BpRes             []int32
	NFragRes          int32
	FragRes           []int32
	Footer            Footer
	bodyIndexesBuffer map[string]*bodyIndexBuffer
	bufferMux         *sync.Mutex
	useBuffer         bool
}

const (
	MAGIC = "HIC"
)

type bodyIndexBuffer struct {
	body  *Body
	count int
	date  time.Time
}
type bodyIndex struct {
	key   string
	count int
	date  time.Time
}
type bodyIndexSlice []bodyIndex

func (p bodyIndexSlice) Len() int {
	return len(p)
}

func (p bodyIndexSlice) Less(i, j int) bool {
	return p[i].date.Before(p[j].date)
}

func (p bodyIndexSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (e *HiC) Lock() {
	e.mutex.Lock()
}
func (e *HiC) Unlock() {
	e.mutex.Unlock()
}
func (e *HiC) Read(p []byte) (int, error) {
	return e.Reader.Read(p)
}
func (e *HiC) Seek(offset int64, w int) (int64, error) {
	return e.Reader.Seek(offset, w)
}
func (e *HiC) LoadBodyIndex(key string) (*Body, error) {
	return e.loadBodyIndex(key)
}
func (e *HiC) loadBodyIndex(key string) (*Body, error) { //loadBodyIndex if it is not in buffer
	//e.Footer.NEntrys[key]
	body, ok := e.bodyIndexesBuffer[key]
	if ok {
		/*
			e.bufferMux.Lock()
			e.bodyIndexesBuffer[key].count += 1
			e.bodyIndexesBuffer[key].date = time.Now()
			e.bufferMux.Unlock()
		*/
		return body.body, nil
	}
	/*
		if len(e.bodyIndexesBuffer) > maxBufferSize {
			go func() {
				e.bufferMux.Lock()
				dateSortedBuffer := make(bodyIndexSlice, 0, len(e.bodyIndexesBuffer))
				for k, d := range e.bodyIndexesBuffer {
					dateSortedBuffer = append(dateSortedBuffer, bodyIndex{k, d.count, d.date})
				}
				sort.Sort(dateSortedBuffer)
				l := len(dateSortedBuffer)
				for i := 0; i < l/3; i++ {
					if i < l/3 {
						delete(e.bodyIndexesBuffer, dateSortedBuffer[i].key)
					}
				}
				e.bufferMux.Unlock()
			}()
		}
	*/
	err := errors.New("init")
	c2 := make(chan Body, 1)
	go func() {
		b := Body{}
		v, ok := e.Footer.Entry[key]
		if !ok {
			err = errors.New("key not found")
			c2 <- b
		}
		e.mutex.Lock()
		e.Seek(v.Position, 0)
		b.Chr1Idx, _ = ReadInt(e)
		b.Chr2Idx, _ = ReadInt(e)
		b.NRes, _ = ReadInt(e)
		b.Mats = make([]BlockMatrix, b.NRes)
		for i := int32(0); i < b.NRes; i++ {
			unit, _ := ReadString(e)
			resIdx, _ := ReadInt(e)
			sumCounts, _ := ReadFloat32(e)
			occupiedCellCount, _ := ReadFloat32(e)
			stdDev, _ := ReadFloat32(e)
			percent95, _ := ReadFloat32(e)
			binSize, _ := ReadInt(e)
			blockBinCount, _ := ReadInt(e)
			blockColumnCount, _ := ReadInt(e)
			blockCount, _ := ReadInt(e)
			blockIndexes := make(map[int]BlockIndex)
			for j := int32(0); j < blockCount; j++ {
				blockID, _ := ReadInt(e)
				blockPosition, _ := ReadLong(e)
				blockSize, _ := ReadInt(e)
				blockIndexes[int(blockID)] = BlockIndex{blockID, blockPosition, blockSize}
			}

			r := (e.Chr[b.Chr1Idx].Length)/binSize + 1
			c := (e.Chr[b.Chr2Idx].Length)/binSize + 1
			//fmt.Println(b.Chr1Idx, b.Chr2Idx, "rc", r, c)
			b.Mats[i] = BlockMatrix{unit, resIdx, sumCounts, occupiedCellCount, stdDev, percent95, binSize, blockBinCount, blockColumnCount, blockCount, blockIndexes, make(map[int]*Block), make(map[int]time.Time), sync.Mutex{}, e, int(r), int(c), e.useBuffer}
			//Not suitable for parrel Mats accessing now.
		}
		e.bodyIndexesBuffer[key] = &bodyIndexBuffer{
			&b,
			1,
			time.Now(),
		}
		e.mutex.Unlock()
		err = nil
		c2 <- b
	}()
	b0 := <-c2
	return &b0, err
}

//Entrys list of chr_chr entrys in hic file
func (e *HiC) Entrys() []string {
	var keys []string
	for k := range e.Footer.Entry {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
func (e *HiC) String() string {
	var s bytes.Buffer
	s.WriteString(fmt.Sprintf("Version: %d\n", e.Version))
	s.WriteString(fmt.Sprintf("Genome: %s\n", e.Genome))
	s.WriteString(fmt.Sprintf("Chromosome Number: %d\n", e.NChrs))
	for i := 0; i < int(e.NChrs); i++ {
		s.WriteString(fmt.Sprintf("\t%s\t%d\n", e.Chr[i].Name, e.Chr[i].Length))
	}
	s.WriteString(fmt.Sprintf("Basepair Resolustions Number : %d\n", e.NBpRes))
	s.WriteString(fmt.Sprintln(e.BpRes))
	s.WriteString(fmt.Sprintf("Fragment Resolustions Number : %d\n", e.NFragRes))
	s.WriteString(fmt.Sprintln(e.FragRes))
	s.WriteString(e.Footer.String())
	return s.String()
}

func NewHiC(useBuffer bool) HiC {
	return HiC{bodyIndexesBuffer: make(map[string]*bodyIndexBuffer), useBuffer: useBuffer}
}

func (e *HiC) getNormalizedVector(key string) ([]float64, error) {
	e.Footer.NormVector[key].Load(e)
	return e.Footer.NormVector[key].Data()
}

//TODO getNormalizedSubMatrix ???
func (e *HiC) readFooter() {
	c := make(chan bool, 1)
	go func() {
		e.Lock()
		e.Seek(e.masterIndexPos, 0)
		e.Footer.NBytes, _ = ReadInt(e)
		e.Footer.NEntrys, _ = ReadInt(e)
		e.Footer.Entry = make(map[string]Index)
		e.Footer.Units = make(map[int]bool)
		for i := int32(0); i < e.Footer.NEntrys; i++ {
			key, _ := ReadString(e)
			filePosition, _ := ReadLong(e)
			sizeInBytes, _ := ReadInt(e)
			e.Footer.Entry[key] = Index{filePosition, sizeInBytes}
		}
		expectedValuesMap := make(map[string]*ExpectedValueFunc)
		nExpectedValues, _ := ReadInt(e)
		for i := int32(0); i < nExpectedValues; i++ {
			no := normtype.NONE            //normalized type TODO
			unitString, _ := ReadString(e) //unit string TODO
			u := unit.StringToIdx(unitString)
			e.Footer.Units[u] = true
			binSize, _ := ReadInt(e)
			b := strconv.Itoa(int(binSize))
			key := strconv.Itoa(u) + "_" + b + "_" + strconv.Itoa(no)
			nValues, _ := ReadInt(e)
			values := make([]float64, nValues)
			for j := int32(0); j < nValues; j++ {
				values[j], _ = ReadFloat64(e)
			}
			nNormalizationFactors, _ := ReadInt(e)
			normFactors := make(map[int32]float64)
			for j := int32(0); j < nNormalizationFactors; j++ {
				chrIdx, _ := ReadInt(e)
				normFactor, _ := ReadFloat64(e)
				normFactors[chrIdx] = normFactor
			}
			df := NewExpectedValueFunc(no, u, int(binSize), values, normFactors)
			expectedValuesMap[key] = df
			// TODO dataset.setExpectedValueFunctionMap(expectedValuesMap)

		}
		if e.Version >= 6 {
			nExpectedValues, err := ReadInt(e)
			if err == nil {
				// No normalization vectors

				for i := int32(0); i < nExpectedValues; i++ {
					typeString, _ := ReadString(e)
					t := normtype.StringToIdx(typeString)
					unitString, _ := ReadString(e)
					//TODO  HiC.Unit unit = HiC.valueOfUnit(unitString);
					u := unit.StringToIdx(unitString)
					binSize, _ := ReadInt(e)
					b := strconv.Itoa(int(binSize))
					key := unitString + "_" + b + "_" + strconv.Itoa(t)
					nValues, _ := ReadInt(e)
					values := make([]float64, nValues)
					for j := int32(0); j < nValues; j++ {
						values[j], _ = ReadFloat64(e)
					}
					nNormalizationFactors, _ := ReadInt(e)
					normFactors := make(map[int32]float64)
					for j := int32(0); j < nNormalizationFactors; j++ {
						chrIdx, _ := ReadInt(e)
						normFactor, _ := ReadFloat64(e)
						normFactors[chrIdx] = normFactor
						//fmt.Println(chrIdx, normFactor)

					}

					//NormalizationType type = NormalizationType.valueOf(typeString);
					df := NewExpectedValueFunc(t, u, int(binSize), values, normFactors)
					expectedValuesMap[key] = df

				}
				nEntries, _ := ReadInt(e)
				normVector := make(map[string]*NormVector)
				normTypes := make(map[int]bool)
				for i := int32(0); i < nEntries; i++ {
					typeString, _ := ReadString(e)
					t := normtype.StringToIdx(typeString)
					chrIdx, _ := ReadInt(e)
					unitString, _ := ReadString(e)
					u := unit.StringToIdx(unitString)
					e.Footer.Units[u] = true
					resolution, _ := ReadInt(e)
					filePosition, _ := ReadLong(e)
					sizeInBytes, _ := ReadInt(e)
					index := &Index{filePosition, sizeInBytes} //TODO IMPROVE TO NormVector
					vector := NewNormVector(t, int(chrIdx), u, int(resolution), index)
					normVector[vector.Key()] = vector
					normTypes[t] = true
				}
				e.Footer.NormVector = normVector
				e.Footer.NormTypes = normTypes
			}
		}
		e.Footer.ExpectedValueMap = expectedValuesMap
		e.Unlock()
		c <- true
	}()
	<-c
	return
}
