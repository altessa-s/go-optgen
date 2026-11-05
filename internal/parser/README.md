# parser

```go
import "github.com/altessa-s/go-optgen/internal/parser"
```

Package `parser` extracts opt-tagged declarations without applying plugin policies.

## Functions

| API | Purpose |
|-----|---------|
| `FindOptFields` | Extracts fields and imports; an optional `TypeResolver` supplies exact `TypeFacts` |
| `ParseOptGen / ParseExprList` | Parse tag declarations |
| `FindNormalizationMethod` | Finds a normalization method |

## Documentation directives

The parser reads `//optgen:doc <Option> <text>` from raw field comment lines before `CommentGroup.Text()` removes Go directives. It stores each option's
lines in `OptField.OptionDoc` and keeps the ordinary `OptField.Doc` clean. Lines for one option retain their order, including empty lines and leading
tabs; unknown option names return a field-scoped parse error. Function-prefix validation is deferred to rendering, after semantic option prefixes are applied.

## Usage

```go
result, err := parser.FindOptFields(pkg, "options", true, nil)
```

See [architecture](../../docs/architecture.md) for dependency rules and [plugin API](../../plugin/README.md) for extension contracts.
