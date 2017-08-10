package bbi

type BedBbiQueryType struct {
	*BedBbiSummaryRecord
	Error error
}

func NewBedBbiQueryType() *BedBbiQueryType {
	return &BedBbiQueryType{NewBedBbiSummaryRecord(), nil}
}

type BedBbiSummaryRecord struct {
	ChromId int
	From    int
	To      int
	Valid   int
	Sum     float64
}

func NewBedBbiSummaryRecord() *BedBbiSummaryRecord {
	record := BedBbiSummaryRecord{}
	return &record
}
func (record *BedBbiSummaryRecord) AddRecord(x BbiSummaryRecord) {
	record.Valid += x.Valid
	record.Sum += x.Sum
}

func (record *BedBbiSummaryRecord) AddValue(x float64) {
	record.Valid += 1
	record.Sum += x
}
