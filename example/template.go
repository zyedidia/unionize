package main

//go:generate unionize -output=union.go Template template.go
type Template struct {
	i1 uint32
	i2 uint16
}
