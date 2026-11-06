# check

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/check"
```

Package `check` implements value-validation plugins.

## Plugins

| API | Purpose |
|-----|---------|
| `RequiredCheck / NonZeroCheck` | Presence and nonzero validation |
| `MinLenCheck / MaxLenCheck` | Length bounds |
| `OneOfCheck` | Allowed string values or identifiers on comparable fields |
| `MinCheck` | Numeric and `time.Duration` lower bound |

## Usage

```go
err := check.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.

`oneof` keeps unquoted values as string literals for the built-in `string` type. On other comparable fields, quoted values remain literals and
unquoted values must be identifiers or qualified constants. `min` bounds are normalized through `DefaultNormalizer`, including duration values such as
`1s`.
