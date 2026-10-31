# plugin

```go
import "github.com/altessa-s/go-optgen/plugin"
```

Package `plugin` defines the public extension API for the optgen code generator. It lives outside `/internal` so that programs embedding the
generator can implement extensions with stable interfaces; the `optgen` CLI registers only the built-in plugins.

## Plugin types

| Interface          | Description                                                                               |
|--------------------|-------------------------------------------------------------------------------------------|
| `FieldPlugin`      | Selects and generates the `WithXxx` function for a field (primary plugin type)            |
| `TransformPlugin`  | Applies a single `optval` modifier (e.g. `lower`, `trimspaces`) to an expression         |
| `ModifierPlugin`   | Pipeline modifier selected by `optgen`/`optval` keys (guards, post-processing)           |
| `TypeDefaultPlugin`| Supplies a default-value expression for a field type when none is specified               |
| `DefaultNormalizer`| Optional `TypeDefaultPlugin` extension: rewrites an explicit tag default into Go code      |
| `CheckPlugin`      | Generates validation code for an `optcheck` key (e.g. `required`, `minlen`)              |
| `PluginMeta`       | Provides `Meta() Meta`; embedded by every plugin contract                                 |

## Pipeline phases

| Phase          | Order | Description                                                                          |
|----------------|-------|--------------------------------------------------------------------------------------|
| Guard          | 1     | Pre-assignment checks (e.g. `notnil`)                                                |
| Transform      | 2     | Value transformations (e.g. `lower`, `trimspaces`, `trimprefix`)                     |
| PostProcess    | 3     | Final cleanup after transforms (e.g. `dedup`, `positive`, `nonempty`)                |
| Check          | 4     | Validation (e.g. `required`, `minlen`, `maxlen`, `oneof`)                            |

## Registration

Create a fresh registry per invocation with `NewRegistry(WithWarningSink(...))` and register implementations explicitly; disable plugins on that
instance with `DisablePlugins`, which rejects unknown names.

```go
func Register(registry *plugin.Registry) error {
    return registry.Register(&MyFieldPlugin{})
}
```

`Register` returns an error wrapping `ErrUnsupportedPlugin`, `ErrAmbiguousPlugin`, `ErrDuplicateKey`, or `ErrDuplicateName` instead of overwriting
or panicking. `optgen`/`optval` keys belong to exactly one transform or modifier plugin, `optcheck` keys and aliases to exactly one check plugin, and
plugin names are unique because they address plugins for disabling.

`init()` registration is not supported.
`FieldPlugin.Generate` receives the value pipeline as an argument. `OutputVar` propagates through all phases and into assignment. `Final` terminates
generation; conditional runtime guards leave it false. See [architecture](../docs/architecture.md).

## Checks

`CheckPlugin.Validate` runs during semantic preparation and rejects a field type the check cannot handle (`ErrNotApplicable`) or an unusable value
(`ErrInvalidCheckValue`), so a declared check never silently generates nothing. A check of a disabled plugin is dropped with a warning.
`ValueKindOf` classifies field types for checks; `BuildNilCheck` emits a reflect-based nil test that compiles for any type.

## Constructor options

| Constructor | Generated options |
|-------------|-------------------|
| `NewCheckBase(key, priority, opts...)` | `WithRequiresValue(bool)`, `WithAliases(...)` |
| `NewTransformBase(key, opts...)` | `WithApplicableTypes(...)`, `WithDefaultForTypes(...)`, `WithDisabledBy(...)`, `WithPriority(...)` |
| `NewRegistry(opts...)` | `WithWarningSink(func(string))` |

## Key types

Pipeline execution is separate from registration: construct it with `plugin.NewPipeline(registry)`.


| Type               | Description                                                                              |
|--------------------|------------------------------------------------------------------------------------------|
| `Registry`         | Per-session registry for all plugin types with priority-based lookup                     |
| `Descriptor`       | Read-only snapshot of a registered plugin returned by `Registry.Descriptors`             |
| `ValueKind`        | Field value classification used by checks (`ValueKindOf`)                                |
| `Meta`             | Plugin metadata: name, kind, priority, and applicable types                              |
| `GenerationContext`| Per-field context passed to plugins with struct info, imports, and helpers                |
| `GeneratedCode`    | Output of `FieldPlugin.Generate`: code, type definitions, and helper functions           |
| `Phase` | Modifier execution phase: guard, transform, post-process, or check |
| `ModifierResult` | Output of `ModifierPlugin.Generate`: code lines, output variable, and termination flag |
| `PipelineResult` | Combined pipeline code, final value variable, and early-return flag |
| `CheckSpec` | Validation key, parameter requirement, and ordering priority |
| `CheckBase` | Embeddable implementation of check metadata, specifications and aliases; configured with `CheckOption` |
| `TransformBase` | Embeddable transform metadata and key implementation; configured with `TransformOption` |
