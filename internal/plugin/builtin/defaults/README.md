# defaults

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/defaults"
```

Package `defaults` provides default expressions during semantic preparation.

## Plugins

| API | Purpose |
|-----|---------|
| `LoggerDefaultPlugin` | slog.New(slog.DiscardHandler) |
| `MapDefaultPlugin` | An initialized map |
| `DurationDefaultPlugin` | Rewrites `time.Duration` tag defaults such as `30s` into `30 * time.Second` (`DefaultNormalizer`) |
| `ConfiguredDefaultPlugin` | Defaults configured per type (`type-defaults` in `.optgen.yaml`); registered by the CLI when configured |

## Usage

```go
err := defaults.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.
