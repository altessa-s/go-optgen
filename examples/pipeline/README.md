# pipeline

```go
import "github.com/altessa-s/go-optgen/examples/pipeline"
```

Two option types generated in one run, configured by a package-local [.optgen.yaml](./.optgen.yaml).

## Generation

```go
//go:generate optgen generate --type checkOptions,transformOptions
```

| Struct             | Option type       | File                        | Constructor           |
|--------------------|-------------------|-----------------------------|-----------------------|
| `checkOptions`     | `CheckOption`     | `check_options_gen.go`      | `newCheckOptions`     |
| `transformOptions` | `TransformOption` | `transform_options_gen.go`  | `newTransformOptions` |

## Configuration

| Setting                          | Effect                                                                                  |
|----------------------------------|-----------------------------------------------------------------------------------------|
| `defaults-from: var`             | Default functions copy `defaultCheckOptionsValue` / `defaultTransformOptionsValue`, then apply `default=` tags |
| `check-failure: skip`            | An argument that fails an `optcheck` rule is dropped; the previous value stays          |
| `slice-setter: slice`            | `WithForbidden([]string)` instead of `WithForbidden(...string)`                         |
| `type-defaults`                  | Every `time.Duration` without a `default=` tag starts at `30 * time.Second`             |

The nearest `.optgen.yaml` wins, so this file replaces the repository one for this package.

## Usage

```go
timeout, err := pipeline.Check(text, pipeline.WithMaxLen(120))
out := pipeline.Transform(text, pipeline.WithLetterCase("upper"))
```
