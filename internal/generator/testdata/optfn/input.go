package optfn

import fmtAlias "fmt"
import clock "time"
import supportAlias "optfn.test/support"

var _ = fmtAlias.Sprint

type currentValue struct{ name string }
type Box[T any] struct{ value T }
type Mapping map[string]string
type ValueMapping map[string]string

func (m *Mapping) WithExact(values map[string]string) Mapping {
	if *m == nil {
		*m = make(Mapping)
	}
	for key, value := range values {
		(*m)[key] = value
	}
	return *m
}

func (m ValueMapping) WithValues(values map[string]string) ValueMapping {
	for key, value := range values {
		m[key] = value
	}
	return m
}

func makeLabel(value string) string                   { return value }
func convertCurrent(value currentValue) currentValue  { return value }
func makeBox[T any](value T) Box[T]                   { return Box[T]{value: value} }
func identity[T any](value T) T                       { return value }
func makeText(value string) (string, error)           { return value, nil }
func echoTime(value clock.Time) clock.Time            { return value }
func makeWidget(value string) supportAlias.Widget     { return supportAlias.Widget{Name: value} }
func acceptCollisions(o, value, err, _ string) string { return o + value + err }

type options[T any] struct {
	// Multi target documentation is retained for each generated setter.
	joined    string              `optfn:"Joined=fmtAlias.Sprint"`
	local     string              `optfn:"Local=makeLabel"`
	mapping   Mapping             `optfn:"Mapping=.WithExact"`
	values    ValueMapping        `optfn:"Values=.WithValues"`
	multi     string              `optgen:"default=\"fallback\"" optfn:"First=makeLabel, Second=fmtAlias.Sprint"`
	box       Box[int]            `optfn:"Build=makeBox[int]"`
	current   currentValue        `optfn:"Current=convertCurrent"`
	stamp     clock.Time          `optfn:"Stamp=echoTime"`
	item      T                   `optfn:"Identity=identity[T]"`
	collision string              `optfn:"Collision=acceptCollisions"`
	widget    supportAlias.Widget `optgen:"default=supportAlias.Widget{}" optfn:"Widget=makeWidget"`
	set       optionsFieldSet     `opt:"-"`
}
