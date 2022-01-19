[![Go](https://github.com/bep/unused/actions/workflows/go.yml/badge.svg)](https://github.com/bep/unused/actions/workflows/go.yml)

This is a small utility that finds _unused exported Go symbols_ (functions, methods ...) in Go. For all other similar use cases, use https://github.com/dominikh/go-tools

I have used this in Hugo (a monorepo with many packages), and it works, but there are some caveats:

* It does not detect references from outside of your project.
* It does not detect references via `reflect`.
* Some possible surprises when it comes to interfaces.

So, you should inspect and test the proposed deletes. See this [test repo](https://github.com/bep/unused-test) for more information.

## Install

```bash
go install github.com/bep/unused@latest
```

You also need `gopls`:

```bash
go install golang.org/x/tools/gopls@latest
```

## Use

`unused` takes only one argument: A [Glob](https://github.com/gobwas/glob) filenam pattern (Unix style slashes, double asterisk is supported) of Go files to check.

Running `unused` in this repository currently gives:

```
unused "**.go"                                                                       
internal/lib/gopls.go:379:2 field Detail is unused (EU1002)
internal/lib/gopls.go:389:2 field Tags is unused (EU1002)
internal/lib/gopls.go:395:2 field Deprecated is unused (EU1002)
internal/lib/gopls.go:401:2 field Range is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:7:2 variable UnusedVar is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:12:2 constant UnusedConst is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:19:6 function UnusedFunction is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:25:2 field UnusedField is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:32:15 method (MyType).UnusedMethod is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:36:6 interface UnusedInterfaceWithUsedAndUnusedMethod is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:38:2 method UnusedInterfaceMethodReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:37:2 method UsedInterfaceMethodReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:41:6 interface UnusedInterface is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:42:2 method UnusedInterfaceReturningInt is unused (EU1002)
internal/lib/testpackages/firstpackage/code1.go:45:6 interface UsedInterface is unused (EU1002
```

Note that we currently skip checking test code, but you do warned about unused symbols only used in tests (see example above).
