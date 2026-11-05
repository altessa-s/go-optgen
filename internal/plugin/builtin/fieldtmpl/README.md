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
| `BaseData` | Field and option type identity, pre-closure snapshot code, ordinary and complete option docs |
| `Data`     | `BaseData` plus pipeline code, final value, and termination flag   |
| `NewData`  | Runs the value pipeline for a field and builds `Data`              |

## Usage

```go
var setter = fieldtmpl.Must("setter", `...{{template "optionReturn" .}}`)

func (p *Plugin) Generate(ctx plugin.GenerationContext, field model.OptField, values plugin.Pipeline) (plugin.GeneratedCode, error) {
	data := fieldtmpl.NewData(ctx, field, values, "v")
	return fieldtmpl.Execute(setter, &data)
}
```

See [architecture](../../../../docs/architecture.md) for dependency rules and [plugin API](../../../../plugin/README.md) for extension contracts.

`Execute` applies a full function doc from `OptField.OptionDoc` or from a field doc that starts with the final function name plus a space. It validates
directive text after the function and semantic option prefixes are known. With neither form, templates render the original generated summary and field
doc paragraph unchanged.
