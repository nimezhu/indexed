package hic

type Chr struct {
	Name   string
	Length int32
}

type Index struct {
	Position int64
	Size     int32
}
type BlockIndex struct {
	Id       int32
	Position int64
	Size     int32
}
