package indexed

import "testing"

func TestHead(t *testing.T) {
	fn := "https://www.encodeproject.org/files/ENCFF609KNT/@@download/ENCFF609KNT.bigWig"
	t.Log(Magic(fn))
	fn2 := "http://genome.compbio.cs.cmu.edu:9000/hg19/hic/K562_combined_30.hic"
	t.Log(Magic(fn2))
}
