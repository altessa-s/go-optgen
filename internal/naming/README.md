# naming

```go
import "github.com/altessa-s/go-optgen/internal/naming"
```

Package `naming` derives Go option names while preserving known initialisms.

## Functions

| API | Purpose |
|-----|---------|
| `OptionName` | Returns HTTPClient for httpClient |

## Usage

```go
name := naming.OptionName("httpClient")
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
