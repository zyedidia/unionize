package main

import (
	"fmt"
	"unsafe"
)

func main() {
	var u TemplateUnion
	u.i1Put(0xdeadbeef)
	u.i2Put(0xface)

	fmt.Printf("i1: 0x%x\n", u.i1()) // prints 0xdeadface
	fmt.Printf("i2: 0x%x\n", u.i2()) // prints 0xface

	fmt.Println("size:", unsafe.Sizeof(u))   // prints 4
	fmt.Println("align:", unsafe.Alignof(u)) // prints 4
}
