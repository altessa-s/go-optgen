# optgen

```go
import "github.com/altessa-s/go-optgen/examples/customplugin/cmd/optgen"
```

The `optgen` CLI with the [myplugins](../../myplugins) plugins registered after the built-in ones; it accepts every flag of the regular binary.

## Usage

| Command                                  | Description                                    |
|------------------------------------------|------------------------------------------------|
| `go run ./cmd/optgen generate ...`       | Generate options with the custom plugins       |
| `go run ./cmd/optgen list-plugins`       | List built-in and custom plugins               |
