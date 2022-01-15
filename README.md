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

`unused` takes only one argument: A [Glog](https://github.com/gobwas/glob) filenam pattern (Unix style slashes, double asterisk is supported) of Go files to check.

Running `unused` in this repository currently gives:

```
unused "**.go"                                                                       
internal/lib/gopls.go:23:2 variable UnusedVar is unused (EU1002)
internal/lib/gopls.go:25:2 variable OnlyUsedInTestVar is used in test only (EU1001)
main.go:154:6 struct UnusedStruct is unused (EU1002)
```

Note that we currently skip checking test code, but you do warned about unused symbols only used in tests (see example above).