# devtools

```go
import "github.com/altessa-s/go-optgen/devtools"
```

Package `devtools` pins development tools in an isolated Go module.

## Tools

| API | Purpose |
|-----|---------|
| `gci / golangci-lint` | Formatting and static analysis |
| `gosec / govulncheck` | Security checks |

## Usage

```go
go install github.com/daixiang0/gci
```

See [architecture](../docs/architecture.md) for dependency rules and [plugin API](../plugin/README.md) for extension contracts.
