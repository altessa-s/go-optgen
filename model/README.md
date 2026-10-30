# model

```go
import "github.com/altessa-s/go-optgen/model"
```

Package `model` defines the public data structures shared between optgen's parser, generator, and plugin subsystems. External plugins should import this
package rather than any internal package to access the types they receive and produce.

## Key types

| Type              | Description                                                                                |
|-------------------|--------------------------------------------------------------------------------------------|
| `RenderSettings` | Caller-chosen names and switches of a generated file: function prefixes, `StringSetterMode`, `KeepEmptyStrings`, `CheckFailureMode` |
| `GenerateInput` | Prepared rendering request passed across the renderer port: `RenderSettings` plus package declarations |
| `SourcePackage` | Syntax extraction result before semantic preparation |
| `OptField`        | Central type capturing everything the parser extracts from a single struct field            |
| `GenericInfo`     | Type parameter information for generic option structs                                      |
| `ImportInfo`      | Resolved package import found in the source file being parsed                              |
| `GeneratedImport` | Import to be rendered in the generated output file                                         |
| `TypeParam`       | A single type parameter (`Name`, `Constraint`) of a generic option struct                   |

## OptField fields

| Field          | Description                                                                                   |
|----------------|-----------------------------------------------------------------------------------------------|
| `FieldName`    | Original field name in the struct                                                             |
| `OptionName`   | Name for the `WithXxx` function                                                               |
| `Type`         | Field type as a string representation                                                         |
| `Default`      | Default value expression from the `optgen` tag or a registered type-default plugin                                                |
| `TypeImports` | Typed import paths and aliases resolved by syntax extraction |
| `BareChecks` | Check keys declared without an explicit value |
| `Doc`          | Documentation comment from the field                                                          |
| `ElemType`     | Element type for slices (without `[]`)                                                        |
| `Modifiers`    | Merged automatic and explicit `optval` modifiers, excluding disablers (e.g. `["trimspaces", "lower"]`)                         |
| `Metadata`     | Flags and `key=value` pairs from the `optgen` tag (e.g. `append`, `notnil`)                   |
| `Checks`       | Validation rules from the `optcheck` tag (e.g. `{"required":"", "minlen":"3"}`)               |
| `IsSlice`      | Whether the field is a slice type                                                             |
| `IsInterface`  | Whether the field is an interface type                                                        |
| `IsNilable`    | Whether the field type can be nil (pointer, slice, map, interface, chan, func)                 |
| `TypeResolved` | Type checking resolved the type, so the type facts are exact rather than inferred from syntax |
| `IsNumeric`    | Integer or floating-point underlying type, including named types such as `time.Duration`      |
| `IsComparable` | Values of the type are comparable (resolved types)                                            |
| `ElemComparable` | Slice elements are comparable (resolved element types)                                      |
| `HasModifiers` | Whether the field has any modifiers                                                           |
| `DefaultOnly`  | Include the field in `defaultOptions()` but skip `WithXxx` generation                         |
