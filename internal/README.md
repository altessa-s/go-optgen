# internal

```go
import "github.com/altessa-s/go-optgen/internal"
```

Package `internal` organizes the private implementation of optgen.

## Subsystems

| API | Purpose |
|-----|---------|
| `application` | Generation use case and ports |
| `parser / semantic` | Syntax extraction and policy application |
| `generator` | Source rendering |
| `source / output` | Filesystem and process adapters |

## Usage

```go
optgen.New()
```

See [architecture](../docs/architecture.md) for dependency rules and [plugin API](../plugin/README.md) for extension contracts.
