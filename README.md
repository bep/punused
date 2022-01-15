This is a small utility that finds _unused exported Go symbols_ (functions, methods ...) in Go. For all other similar use cases, use https://github.com/dominikh/go-tools

I have used this in Hugo (a monorepo with many packages), and it works. But isn't particulary fast (it uses `gopls` CLI, I should look into running that as a language server). and there are some caveats:

* It does not detect references from outside of your project. TODO(bep) figure out what the search path is.
* It does not detect references via `reflect`.
* It does not detect references from test files.

So, you should inspect and test the proposed deletes.

## Install


```bash
go install github.com/bep/unused@latest
```

You also need `gopls`:

```bash
go install golang.org/x/tools/gopls@latest
```


