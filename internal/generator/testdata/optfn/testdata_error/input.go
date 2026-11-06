package optfn

func makeText(value, err string) (string, error) { return value, nil }

type options struct {
	result string          `optfn:"Result=makeText"`
	set    optionsFieldSet `opt:"-"`
}
