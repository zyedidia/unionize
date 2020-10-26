This is the example from the main readme. First make sure `unionize` is in your
`PATH`. Then run:

```
go generate
go build
./example
```

Note that the generation command is `unionize Template template.go`. The
package is `template.go` instead of `.` because the `.` package has a
compilation error before generation happens (because it uses the
`TemplateUnion` type), so it cannot be parsed by `unionize`. If your union
needs to use package variables, I suggest generating the union before writing
any code that uses it.
