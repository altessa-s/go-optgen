# fieldtmpl

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin/fieldtmpl"
```

Package `fieldtmpl` provides templates and option data shared by the built-in [field plugins](../fields/README.md).

## Key types

| API        | Purpose                                                            |
|------------|--------------------------------------------------------------------|
| `Must`     | Parses a field template with the shared `optionReturn` definition  |
| `Execute`  | Renders one option function                                        |
| `BaseData` | Field and option type identity                                     |
| `Data`     | `BaseData` plus pipeline code, final value, and termination flag   |
| `NewData`  | Runs the value pipeline for a field and builds `Data`              |

## Usage

```go
var setter = fieldtmpl.Must("setter", `...{{template "optionReturn" .}}`)

func (p *Plugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	return fieldtmpl.Execute(setter, fieldtmpl.NewData(ctx, field, values, "v"))
}
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.
