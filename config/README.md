# config

```go
import "github.com/altessa-s/go-optgen/config"
```

Package `config` decodes and merges `.optgen.yaml` configuration documents for the optgen code generator. It performs no I/O: the optgen command
finds the file (the first `.optgen.yaml` between the target directory and the project boundary, `go.mod` / `.git`) and passes its contents to `Parse`.

## Functions

| Function | Description                                                 |
|----------|-------------------------------------------------------------|
| `Parse`  | Decode the contents of an `.optgen.yaml` file into a `File` |

## Key types

| Type             | Description                                                                                 |
|------------------|---------------------------------------------------------------------------------------------|
| `File`         | Top-level configuration with `Defaults` and per-package `Packages` map                      |
| `Defaults`       | Global defaults: type name, output file, option type, formatter, plugin lists               |
| `PackageConfig`  | Per-package overrides keyed by relative path; unset fields inherit from `Defaults`           |

## Defaults fields

| Field             | Description                                                                                |
|-------------------|--------------------------------------------------------------------------------------------|
| `Type`            | Struct type name to scan for option fields                                                 |
| `Output`          | Output file name for generated code                                                        |
| `OptionType`      | Name of the generated `Option` type                                                        |
| `OptionError`     | Generate `Option` as `func(*T) error` instead of `func(*T)`                                |
| `Formatter`       | Formatter command; unset keeps `gofmt -w`, an empty string disables formatting             |
| `DisablePlugins`  | List of plugin names to disable                                                            |
| `AllFields`       | Process every struct field, not only tagged ones (`all-fields`)                            |
| `SkipOptionType`  | Skip generating the `Option` type definition                                               |
| `SkipDefaultFunc` | Skip generating the `defaultOptions` function                                              |
| `SkipNewFunc`     | Skip generating the `newOptions` function                                                  |
| `DefaultFuncName` | Custom name for the default function                                                       |
| `NewFuncName`     | Custom name for the new function                                                           |
| `OptionPrefix`    | Prefix for generated option function names                                                 |
| `Verbose`         | Enable verbose output                                                                      |
| `NoFormat`        | Disable formatting of generated files                                                      |
| `FuncPrefix`      | Option function prefix (`func-prefix`, default `With`)                                     |
| `InvertFuncPrefix`| Prefix of inverted bool flag options (default `Without`)                                   |
| `StringSetter`    | String option signature: `generic` (default) or `plain`                                    |
| `KeepEmptyStrings`| Assign empty strings instead of skipping them                                              |
| `CheckFailure`    | Failed check without an error result: `panic` (default) or `skip`                          |
| `NoDefaultModifiers` | Do not apply the modifiers plugins add by default                                       |
| `Validate`, `WithIsSet`, `Strict` | `--validate`, `--with-isset` and `--strict` |
| `DefaultsFrom`, `SliceSetter`, `Exclude` | `--defaults-from`, `--slice-setter` and `--exclude` |
| `TypeDefaults`    | Default expression per field type, e.g. `"*slog.Logger": slog.Default()`; packages merge per key |

YAML keys are kebab-case (`option-type`, `all-fields`, `disable-plugins`). `File.GetPackageConfig(pkgPath)` returns the `Defaults` merged with the
`packages` override for that path.

External plugins are not supported: `Parse` rejects a `defaults.plugins` list with `ErrPluginsUnsupported`.
