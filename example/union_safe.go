// +build safe

// Code generated by unionize.
package main

type TemplateUnion struct {
	_i1 uint32
	_i2 uint16
}

func (u *TemplateUnion) i1() uint32 {
	return u._i1
}
func (u *TemplateUnion) i1Put(v uint32) {
	u._i1 = v
}

func (u *TemplateUnion) i2() uint16 {
	return u._i2
}
func (u *TemplateUnion) i2Put(v uint16) {
	u._i2 = v
}
