# examples

Runnable packages that generate their options with `optgen`. Each one commits its `options.go`, the generated files, and `Example` functions whose
`// Output:` is checked by `go test`, so the examples cannot drift from the generator: `make generate` regenerates them with the rest of the
repository.

| Example                          | Shows                                                                                                            |
|----------------------------------|------------------------------------------------------------------------------------------------------------------|
| [server](./server)               | `--option-error`, `--validate`, every built-in check, `mandatory`, `parseip` with `err=`, `append`, `invert`, `param` |
| [cache](./cache)                 | A generic options struct, explicit type arguments, `--with-isset`                                                |
| [pipeline](./pipeline)           | Several option types in one run, a package `.optgen.yaml`: `defaults-from: var`, `check-failure: skip`, `slice-setter`, `type-defaults` |
| [customplugin](./customplugin)   | A transform and a check plugin of your own, registered through a wrapping `main`                                 |

Run them all:

```sh
go test ./examples/...
```

Every example except `pipeline` also picks up the repository [.optgen.yaml](../.optgen.yaml), which sets `all-fields: true` and the formatter.
