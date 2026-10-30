# server

```go
import "github.com/altessa-s/go-optgen/examples/server"
```

Strict options: every option returns an error, and the constructor re-checks the `optcheck` rules on the final configuration.

## Generation

```go
//go:generate optgen generate --option-error --validate --source-file $GOFILE
```

## Fields

| Field            | Tags                                                         | Generated                                                       |
|------------------|--------------------------------------------------------------|-----------------------------------------------------------------|
| `addr`           | `optgen:"mandatory"`                                         | First parameter of `newOptions`, no option                      |
| `name`           | `optval:"lower" optcheck:"required,minlen=3,maxlen=32"`      | `WithName` trims, lower-cases, then checks                      |
| `mode`           | `optgen:"default=DefaultMode" optcheck:"oneof=[http,https,h2c]"` | `WithMode` rejects values outside the set                   |
| `readTimeout`    | `optgen:"default=DefaultReadTimeout" optval:"positive"`      | `WithReadTimeout` ignores nonpositive values                    |
| `maxConns`       | `optgen:"default=DefaultMaxConns" optcheck:"nonzero"`        | `WithMaxConns` rejects zero                                     |
| `trustedProxies` | `optgen:"parseip,err=ErrInvalidIP"`                          | `WithTrustedProxies(...string)` returns `ErrInvalidIP`          |
| `tags`           | `optgen:"append" optval:"nonempty,dedup"`                    | `WithTags` appends, dropping empty and duplicate entries        |
| `disableKeepAlive` | `opt:"KeepAlive" optval:"invert"`                          | `WithoutKeepAlive()` sets the field to `true`                   |
| `debug`          | `optval:"param"`                                             | `WithDebug(bool)`                                               |
| `logger`         | none (`all-fields`)                                          | Defaults to a discarding `*slog.Logger`; nil is ignored         |
| `headers`        | none (`all-fields`)                                          | Defaults to an empty map                                        |
| `started`        | `opt:"-"`                                                    | Nothing                                                         |

## Usage

```go
s, err := server.New(":8080",
	server.WithName("api"),
	server.WithTrustedProxies("10.0.0.1"),
	server.WithoutKeepAlive(),
)
```

See [example_test.go](./example_test.go) for the validation errors.
