# builtin

```go
import "github.com/altessa-s/go-optgen/internal/plugin/builtin"
```

Package `builtin` registers every built-in plugin into a per-invocation [registry](../../../plugin/README.md).

## Key types

| API        | Purpose                                                    |
|------------|------------------------------------------------------------|
| `Register` | Installs fresh instances of all built-in plugins in a registry |

## Subpackages

| Package                          | Plugins                                          |
|----------------------------------|--------------------------------------------------|
| [`check`](check/README.md)       | optcheck validations                             |
| [`defaults`](defaults/README.md) | Type default values                              |
| [`fields`](fields/README.md)     | WithXxx field generators                         |
| [`guard`](guard/README.md)       | Early-return guards                              |
| [`postprocess`](postprocess/README.md) | Value post-processors                      |
| [`transform`](transform/README.md) | optval string transforms                       |
| [`fieldtmpl`](fieldtmpl/README.md) | Templates shared by field plugins (not a plugin) |

## Usage

```go
r := plugin.NewRegistry()
if err := builtin.Register(r); err != nil {
	return err
}
```

Generated code imports only the standard library and packages referenced by the source struct.

See [architecture](../../../docs/architecture.md) for dependency rules and [plugin API](../../../plugin/README.md) for extension contracts.
