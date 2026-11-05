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
| `BoolFlagPlugin / NetIPParsePlugin` | Boolean toggles and IP conversion |

## Usage

```go
err := fields.Register(registry)
```

`optfn` targets carry their resolved parameter names, types, and variadic behavior in `model.SetterTarget`. `SetterFuncPlugin` uses those values and
adds the target's imports to the generated file; its input does not pass through the value modifier pipeline.

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.
