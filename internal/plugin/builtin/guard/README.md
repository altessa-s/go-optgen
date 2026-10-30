# guard

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/guard"
```

Package `guard` implements conditional runtime guards.

## Plugins

| API | Purpose |
|-----|---------|
| `NotNilModifier` | Skips nil or empty input without terminating pipeline construction |

## Usage

```go
err := guard.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.

## Generated imports

| Import    | When emitted                                                                                   |
|-----------|------------------------------------------------------------------------------------------------|
| `reflect` | Guards for interfaces and unresolved named types, which must also detect a wrapped nil pointer |
