package bigwig

import (
	"fmt"
	"os"
	"testing"

	. "github.com/nimezhu/netio"
)

func initBwf(bwf *BbiReader, filename string) {
	if err := bwf.Header.Read(bwf.Fptr); err != nil {
		fmt.Errorf("reading `%s' failed: %v", filename, err)
	}
	fmt.Println("reading header")
	if bwf.Header.Magic != BIGWIG_MAGIC {
		fmt.Errorf("reading `%s' failed: not a BigWig file", filename)
	}
	fmt.Println("checking magic")
	// parse chromosome list, which is represented as a tree
	if _, err := bwf.Fptr.Seek(int64(bwf.Header.CtOffset), 0); err != nil {
		fmt.Errorf("reading `%s' failed: %v", filename, err)
	}
	fmt.Println("seekint ot offset")
	if err := bwf.ChromData.Read(bwf.Fptr); err != nil {
		fmt.Errorf("reading `%s' failed: %v", filename, err)
	}
	fmt.Println("reading chromosome")
	// parse data index
	if _, err := bwf.Fptr.Seek(int64(bwf.Header.IndexOffset), 0); err != nil {
		fmt.Errorf("reading `%s' failed: %v", filename, err)
	}
	fmt.Println("seeking to index offset")
	if err := bwf.Index.Read(bwf.Fptr); err != nil {
		fmt.Errorf("reading `%s' failed: %v", filename, err)
	}
	fmt.Println("reading index offset")
	// parse zoom level indices
	bwf.IndexZoom = make([]RTree, bwf.Header.ZoomLevels)
	for i := 0; i < int(bwf.Header.ZoomLevels); i++ {
		fmt.Println("reading zooming level", i)
		if _, err := bwf.Fptr.Seek(int64(bwf.Header.ZoomHeaders[i].IndexOffset), 0); err != nil {
			fmt.Errorf("reading `%s' failed: %v", filename, err)
		}
		if err := bwf.IndexZoom[i].Read(bwf.Fptr); err != nil {
			fmt.Errorf("reading `%s' failed: %v", filename, err)
		}
	}
	fmt.Println("done reading indexes")
}

func initLargeBwf(bwf *BbiReader, filename string) {
	fn := "chipseq.bw.index"
	if _, err := os.Stat(fn); os.IsNotExist(err) {
		bwf.InitIndex()
		f, _ := os.Create(fn)
		bwf.WriteIndex(f)
		f.Close()
	} else {
		fr, _ := os.Open(fn)
		bwf.ReadIndex(fr)
		fr.Close()
	}

	fmt.Println("done reading indexes")
}
func testBwf(bwf *BbiReader, filename string) {
	initBwf(bwf, filename)
	for i := range bwf.Query(0, 0, 40, 10) {
		fmt.Println(i)
	}
	for i := range bwf.Query(0, 0, 200, 50) {
		fmt.Println(i)
	}
	fmt.Println("QueryRaw")
	for v := range bwf.QueryRaw(0, 0, 200) {
		fmt.Println(v)
	}

}
func testEncode(bwf *BbiReader, filename string) {
	fmt.Println("begin reading")
	initLargeBwf(bwf, filename)
	fmt.Println("done reading in encode rnqseq")
	//fmt.Println(bwf.Header)
	//fmt.Println(bwf.ChromData)
	for i := range bwf.Query(2, 0, 100000, 224) {
		fmt.Println("result")
		fmt.Println(i)
	}

}

func TestBW(t *testing.T) {
	f, _ := os.Open("test.bw")
	bwf := NewBbiReader(f)
	bwf.Fptr = f
	t.Log("testing local file")
	testBwf(bwf, "test.bw")
}

var bigwigURI = "http://ftp.ebi.ac.uk/pub/databases/ensembl/encode/integration_data_jan2011/byDataType/rna_signal/jan2011/hub/wgEncodeCshlLongRnaSeqA549CellLongnonpolyaMinusRawSigRep1.bigWig"
var rnaseqURI = "http://genome.compbio.cs.cmu.edu:9000/hg19/bigwig/rnaseq.bw"
var chipseqURI = "http://genome.compbio.cs.cmu.edu:9000/hg19/bigwig/chipseq.bw"
var bigbedURI = "http://ftp.ebi.ac.uk/pub/databases/ensembl/encode/integration_data_jan2011/byDataType/segmentations/jan2011/hub/k562.combined.bb"

func TestReadSeeker(t *testing.T) {
	f, _ := NewReadSeeker("http://genome.compbio.cs.cmu.edu:9000/test/bigwig/test.bw")
	bwf := NewBbiReader(f)
	t.Log("testing httpstream")
	testBwf(bwf, "web")
}
func TestEncode(t *testing.T) {
	f, _ := NewHttpReadSeeker(chipseqURI)
	f.BufferSize(4096)
	bwf := NewBbiReader(f)
	t.Log("testing encode")
	testEncode(bwf, "encode")
}

func TestBigBed(t *testing.T) {
	f, _ := NewHttpReadSeeker(bigbedURI)
	f.BufferSize(8192)
	bwf := NewBbiReader(f)
	bwf.InitIndex()
	for i := uint16(0); i < bwf.Header.ZoomLevels; i++ {
		binsize := int(bwf.Header.ZoomHeaders[i].ReductionLevel)
		t.Log(binsize)
	}
}
