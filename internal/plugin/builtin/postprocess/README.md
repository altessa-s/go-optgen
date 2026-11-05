# postprocess

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/postprocess"
```

Package `postprocess` implements value cleanup after transformation.

## Plugins

| API | Purpose |
|-----|---------|
| `DedupModifier` | Deduplicates comparable slices |
| `CloneModifier` | Copies slice or map values inside the option closure, or snapshots them at option creation with `clone=eager` |
| `NonEmptyModifier` | Skips empty scalars or filters slice elements |
| `PositiveModifier` | Skips nonpositive numeric values |

## Usage

```go
err := postprocess.Register(registry)
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.

## Generated imports

Post-processors emit inline code and add no imports except `CloneModifier`, which uses `slices.Clone` or `maps.Clone`. Deduplication copies into a new
slice, keeping the first occurrence of each element, so the caller's variadic argument is never modified.
