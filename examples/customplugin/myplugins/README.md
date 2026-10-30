# myplugins

```go
import "github.com/altessa-s/go-optgen/examples/customplugin/myplugins"
```

Two optgen plugins built on the public [plugin](../../../plugin) API, registered with `optgen.WithPlugins(myplugins.Register)`.

## Key types

| Type       | Tag                   | Contract                                 | Generated code                                       |
|------------|-----------------------|------------------------------------------|------------------------------------------------------|
| `Collapse` | `optval:"collapse"`   | `TransformPlugin` via `TransformBase`    | `strings.Join(strings.Fields(v), " ")`               |
| `HostPort` | `optcheck:"hostport"` | `CheckPlugin` via `CheckBase`            | `net.SplitHostPort(v)`; fails per `--option-error` / `--check-failure` |

`HostPort.Validate` rejects non-string fields with `plugin.ErrNotApplicable`, so a misplaced tag fails generation instead of emitting code that
does not compile.
