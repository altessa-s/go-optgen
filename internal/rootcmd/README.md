# rootcmd

```go
import "github.com/altessa-s/go-optgen/internal/rootcmd"
```

Package `rootcmd` constructs Cobra roots with version reporting and generated options. `Run` recovers panics during execution.

## Functions

| API | Purpose |
|-----|---------|
| `New(use, short, opts...)` | Constructs the root command |
| `Run(use, short, args, opts...)` | Executes with panic recovery |
| `Version()` | Reports the build or module version |

## Options

| Option | Purpose |
|--------|---------|
| `WithLong`, `WithExample` | Detailed help and examples |
| `WithSubcommands` | Child commands |
| `WithOut`, `WithErr` | Output writers; defaults are standard streams |

See the [root package](../../README.md#library-api) for application assembly.
