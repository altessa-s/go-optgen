# customplugin

```go
import "github.com/altessa-s/go-optgen/examples/customplugin"
```

Generation with plugins of your own: [myplugins](./myplugins) adds the `optval:"collapse"` transform and the `optcheck:"hostport"` check.

## Generation

The `optgen` binary registers only the built-in plugins, so the directive runs [cmd/optgen](./cmd/optgen), a `main` that wraps `optgen.Run`:

```go
//go:generate go run ./cmd/optgen generate --option-error --validate --source-file $GOFILE
```

```go
func main() {
	if err := optgen.Run(os.Args[1:], optgen.WithPlugins(myplugins.Register)); err != nil {
		os.Exit(1)
	}
}
```

## Fields

| Field       | Tags                                      | Generated                                                         |
|-------------|-------------------------------------------|-------------------------------------------------------------------|
| `endpoint`  | `optcheck:"required,hostport"`            | `WithEndpoint` wraps the `net.SplitHostPort` error                |
| `userAgent` | `optval:"collapse" optcheck:"maxlen=64"`  | `WithUserAgent` collapses inner white space, then checks length   |
