# fields

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/fields"
```

Package `fields` implements functional-option field renderers.

## Plugins

| API | Purpose |
|-----|---------|
| `ManualPlugin` | Skips setters while keeping explicit defaults |
| `DefaultSetterPlugin` | Scalar and string setters |
| `SetterFuncPlugin` | Function- or method-backed setters selected by `optfn` |
| `SliceSetPlugin / DefaultAppenderPlugin` | Slice replacement or append |
| `MapSetPlugin` | Replace or accumulate keys in `map[K]struct{}` with `set` |
| `MapPutPlugin` | Put a key/value pair into a map with `put` |
| `BoolFlagPlugin / NetIPParsePlugin` | Boolean toggles and IP conversion |

## Usage

```go
err := fields.Register(registry)
```

`optfn` targets carry their resolved parameter names, types, and variadic behavior in `model.SetterTarget`. `SetterFuncPlugin` uses those values and
adds the target's imports to the generated file; its input does not pass through the value modifier pipeline.

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.

`set` and `put` map fields bypass the value modifier pipeline. `set` accepts `map[K]struct{}` and inserts keys; add `append` to accumulate keys
instead of replacing the map. `put` accepts any map and lazily creates it when adding its first entry.
