# Architecture

The generation use case owns the operation sequence. CLI and filesystem adapters supply its ports; models and plugin contracts never import those
adapters. Required dependencies are explicit constructor parameters and each invocation assembles a fresh plugin registry.

```mermaid
flowchart TD
    CLI[optgen root package: assembly and presentation] --> App[application: generation use case]
    CLI --> Source[source: filesystem and Go syntax]
    CLI --> Semantic[semantic: validation and defaults]
    CLI --> Generator[generator: rendering]
    CLI --> Output[output: formatter and atomic publication]
    App --> Model[model: requests and declarations]
    Source --> Parser[parser: syntax only]
    Semantic --> Plugin[plugin: contracts and session pipeline]
    Generator --> Plugin
    Plugin --> Model
    Parser --> Model
```

## Boundaries

| Package | Responsibility | Allowed inner dependencies |
|---------|----------------|----------------------------|
| `model` | Generation data, field declarations and option docs, all source struct fields and their type facts, import information, resolved setter signatures and map entry facts | Standard library |
| `plugin` | Extension contracts, explicit catalogs, pipeline execution | `model` |
| `internal/application` | Parse, prepare, render, publish through ports | `model` |
| `internal/parser` | Extract syntax without applying plugin policies; resolves `optfn` targets and map entry facts through the type resolver port | `model`, `internal/naming` |
| `internal/semantic` | Validate checks, apply the option prefix, resolve type defaults, default normalization, default modifiers and settings policies (`defaults-from=none`, `with-isset`) | `model`, `plugin` |
| `internal/generator` | Render prepared input; applies no policy | `model`, `plugin` |
| `internal/source` | Load and type-check Go packages (`go/packages`), invoke syntax extraction with type facts and resolved function signatures | `model`, `internal/parser` |
| `internal/output` | Run a formatter and atomically publish successful output | Standard library |
| `internal/naming` | Derive option names from field names | Standard library |
| `internal/plugin/builtin` | Register the built-in plugins | `plugin`, its sub-packages |
| `internal/plugin/builtin/*` | One plugin family each; `fields` may use the `fieldtmpl` template helpers | `model`, `plugin` (`fields`: also `fieldtmpl`) |
| `config` | Decode and merge `.optgen.yaml` documents; no I/O | Standard library, `yaml.v3` |
| `internal/rootcmd` | Cobra root command and version output | Standard library, Cobra |
| `cmd/optgen` | Binary entry point | `optgen` |
| `optgen` (module root) | Assemble implementations, find and read configuration, adapt CLI input, present progress | Inner packages and adapters |

## Decisions

| Decision | Reason |
|----------|--------|
| Per-invocation registry and warning sink | Disabled plugins and diagnostics must not leak between commands. |
| Registration errors instead of overwrites | A key, alias or name owned by two plugins makes one of them silently unreachable. |
| Separate key namespaces | `optgen`/`optval` keys select transforms and modifiers; `optcheck` keys select check plugins only. |
| Checks validate applicability | A check that cannot handle a field type fails preparation instead of generating nothing. |
| Policies derived from descriptors | Default modifiers and disablers are semantic policies computed from `Registry.Descriptors`, not registry queries. |
| Generated code imports the standard library | Generated files must compile in any module, including the altessa-s modules that run optgen. |
| Explicit registration | Importing a package must not change generation behavior. |
| One pipeline | All field handlers share ordering, value propagation, and termination semantics. |
| Syntax before semantics | Parsing remains deterministic regardless of registered extensions. |
| Type facts through a parser port | The parser consumes `TypeResolver` facts and resolved `optfn` signatures; only the source adapter depends on `go/packages`. |
| Plugins compiled in | Third-party plugins register through `optgen.WithPlugins` in a wrapping `main`; nothing is loaded at runtime. |
| Format before publication | Failed formatting preserves the previous generated file. |
| Generated constructor options | Package tunables follow the same option conventions as generated consumers. |
| Settings verified by the use case | `RenderSettings.Verify` runs in `application.Service`, so every caller, not only the CLI, rejects unknown modes. |
| Default normalization through plugins | Type-specific rewriting of tag defaults (e.g. `30s` for `time.Duration`) is a `DefaultNormalizer` hook, not a semantic branch. |
| Check bound normalization through plugins | Numeric check bounds use the field type's `DefaultNormalizer` (e.g. `1s` for `time.Duration`) before check validation and rendering. |
| Cross-field option updates | `enables` and `resets` are validated against `SourcePackage.StructFields` and rendered by shared field templates after assignment. |
| Collection cloning after post-processing | Clone modifiers run after all value transforms and post-processors. Eager snapshots are emitted by field templates before closure creation. |

## Plugin contracts

The pipeline runs guards, transforms, post-processors, checks, and final collection clones in order. `OutputVar` becomes the input to subsequent
modifiers and the final assignment. `Final` stops generation of subsequent modifiers and assignment; it must only be set when the plugin emits a
terminal statement on every path. A conditional `return` leaves `Final` false. Transform plugins run before modifiers, in tag order.
Check plugins run by priority and key after modifier plugins in the check phase. Metadata keys are sorted to keep generation
deterministic. Field plugins receive the value pipeline (`plugin.Pipeline`, `BuildPipeline` only) as a `Generate` argument; check rendering for the
generated validation function is a generator-side interface. `GenerationContext` carries generation data and collects the imports plugins emit
(`AddImport`).

The `optfn` syntax is parsed into one `OptField` per target. The source adapter resolves function expressions with `types.Eval` at the field position
and methods with `types.LookupFieldOrMethod`; the model carries only signature strings, variadic flags, and generated imports. The source adapter keeps
its `token.FileSet` and `types.Package` private. Map fields also carry
rendered key and value types plus value nilability for put setters. Function-backed setters do not build the value pipeline. Defaults are deduplicated
by field before rendering because multiple targets can share one field.

The parser extracts every source field into `SourcePackage.StructFields`, including fields skipped with `opt:"-"`; resolved type facts and import
references allow semantic validation and rendering of `enables` and `resets` targets. The parser also extracts `//optgen:doc` lines from raw comment
groups into the matching expanded `OptField` while leaving `Doc` as ordinary field text. Semantic preparation applies the option prefix to directive
docs, and the generator validates them against the final function name using the render settings for function prefixes. `fieldtmpl` also detects a field
doc that starts with the full generated function name and uses it as the complete doc; multi-option fields discard a doc that starts with any generated
function name so it cannot leak to sibling options. All field plugins share this behavior through their common templates.

The source adapter resolves the types and constant status of check expressions and stores only assignability results on each `OptField`. Check plugins
use those results during semantic preparation to reject bounds and case values that would fail in generated code.

Collection setters bypass value modifiers for map keys and values. `clone=eager` snapshots slice or map input before the option closure is created through
the enabled clone modifier's pipeline result, while `clone` copies during closure application. Disabled clone plugins emit neither snapshots nor imports.

The CLI registers only the built-in plugins; external shared objects are not loaded. Plugin implementations must keep per-generation mutable state
in the generation context, not in package globals or shared plugin instances.

## Verification

`TestDependencyDirection` enforces the import rules of every package in the table, and fails for a package without a rule; `TestNoInitOrDynamicLoading`
rejects `init` functions and the standard library `plugin` package repo-wide. Contract tests cover pipeline substitution, every field handler with and
without error results, and disabled checks; CLI tests verify session isolation. Golden tests review output changes and type-check standard-library
fixtures. Parser fuzz targets check tag round trips and list invariants. Run `make generate`, `go test -race -shuffle=on ./...`, `go vet ./...`, and
`golangci-lint run ./...` after architecture changes.
