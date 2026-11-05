package optiondocs

func makeValue(value string) string { return value }

type options struct {
	// SetName replaces the configured name.
	name string `opt:"Name"`
	// SetPaths replaces all configured paths.
	paths []string `opt:"Paths"`
	// WithoutEnabled disables the option.
	enabled bool `opt:"Enabled" optval:"invert"`
	//optgen:doc Exact SetExact applies the exact mapping.
	//optgen:doc Exact
	//optgen:doc Exact 	value := makeValue(input)
	//optgen:doc Prefix SetPrefix applies the prefix mapping.
	mapping string `optfn:"Exact=makeValue, Prefix=makeValue"`
	// This field doc does not match the generated function name.
	untouched int `opt:"Untouched"`
}
