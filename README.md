# Unionize

This tool provides a way to generate simple C-style unions in Go with no
overhead. Since C-style unions do not exist in Go, implementing them requires
use of code generation, and the unsafe package. Unionize also ensures that the
resulting union type has correct size and alignment. Please only use this
package if you really require high performance for your unions. See the
discussion section for details (alternatives and use-cases).

This tool is new and has not been thoroughly tested. If you find an problem,
please open an issue or pull request.

Use with caution!

# Installation

```
go get github.com/zyedidia/unionize
```

This will install a command-line tool called `unionize` which you can use to
generate unions.

# Usage

First create the union template in your source code:

```go
type Template struct{
    i1 uint32
    i2 uint16
}
```

Run `unionize` and provide the name of the template, plus the package the
template is defined in.

```
unionize -output=template_union.go Template .
```

This will output

```go
package main

import ( ... )

type TemplateUnion struct { ... }

func (u *TemplateUnion) i1() uint32     { ... }
func (u *TemplateUnion) i1Set(v uint32) { ... }

func (u *TemplateUnion) i2() uint16     { ... }
func (u *TemplateUnion) i2Set(v uint16) { ... }
```

By default, `unionize` will print to stdout, but the `-output` option lets you
specify a file as output. See `unionize -h` for additional options for
generation. If you are interested, take a look at the generated code to see how
it works.

Once the union is generated, you can use it in your code:

```go
var u TemplateUnion
u.i1Put(0xdeadbeef)
u.i2Put(0xface)

fmt.Printf("0x%x\n", u.i1()) // prints 0xdeadface
fmt.Printf("0x%x\n", u.i2()) // prints 0xface
```

See `example/` for details. In that example, `go generate` is used to create
the union automatically.

# Discussion

Sum types are not present in Go, primarily because they can be recreated using
interfaces. For example, suppose we wanted to create a stack that can hold
various types of entries simultaneously. We want to be able to have entries
that are of type `A`, `B`, or `C`:

```go
type A struct {
    i int
}

type B struct {
    i1 uint32
    i2 uint32
    s  string
}

type C struct {
    i int
    v interface{}
}
```

The idiomatic Go solution would be to create an interface called `StackEntry`
which includes a dummy function, and define the dummy function for each type
we want to include in the sum type:

```go
type StackEntry interface {
    isStackEntry()
}

func (*A) isStackEntry() {}
func (*B) isStackEntry() {}
func (*C) isStackEntry() {}
```

Now we can add objects of type `A`, `B`, or `C` to the stack, and use type
assertions to determine what type any given entry is.

This has a major downside, which is that interfaces are heap allocated so if we
are constantly pushing and popping the stack, this results in a lot of memory
allocation and GC pressure. Additionally each interface only stores a pointer
to the actual data, so reading the data for an entry involves an additional
random access. This is not always a problem, but sometimes it is.

One solution that is still somewhat idiomatic is to make a giant struct and
pack everything in:

```go
const (
    seA byte = iota
    seB
    seC
)

type StackEntry struct {
    t byte
    a A
    b B
    c C
}
```

We add a marker to know which field is active. This is still pretty clean, but
the size of the StackEntry is now proportional to the number of types we want
it to store, even though only one type can be active at a time! We no longer
have a heap allocation or indirection problem, but the stack entry may be much
larger than it needs to be, resulting in low performance from needing to copy a
lot of unused data, and high memory footprint. If neither of these solutions
work, then we need to reach for the C-style union.

This tool tries to fix these problems with zero-overhead C-style unions in Go.
Here is how we might handle the situation with the stack entries: we define
a template "entry" struct and will use that to generate a union type.

```go
type entry struct {
    a A
    b B
    c C
}

type StackEntry struct {
    t byte // active type indicator
    u entryUnion
}
```

After running `unionize entry .`, we get a file that declares the `entryUnion`
type and can use that.

When we need to create an entry to push to the stack, we just use

```go
var ent StackEntry
ent.t = seA
ent.u.aPut(A{...})

// or

ent.t = seB
ent.u.bPut(B{...})

// or

ent.t = seC
ent.u.cPut(C{...})
```

When we get a union type by popping from the stack, we inspect it like so:

```go
ent := stack.Pop()

switch ent.t {
case seA:
    a := ent.u.a()
    // ...
case seB:
    b := ent.u.b()
    // ...
case seC:
    c := ent.u.c()
    // ...
}
```

This method requires more work and requires code generation, but avoids the
problems of heap allocation, indirection from interfaces, and large structs.
