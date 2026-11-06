package collections

type optionsFieldSet uint64

type options[K comparable] struct {
	Replace     []string         `opt:"Replace" optval:"clone,notrim,nodup"`
	Eager       []string         `opt:"Eager" optval:"clone=eager,notrim,nodup"`
	Append      []string         `opt:"Append" optgen:"append" optval:"clone=eager,notrim,nodup"`
	AppendClone []string         `opt:"AppendClone" optgen:"append" optval:"clone,notrim,nodup"`
	Map         map[string]int   `opt:"Map" optval:"clone"`
	EagerMap    map[string]int   `opt:"EagerMap" optval:"clone=eager"`
	Keys        map[K]struct{}   `opt:"Keys" optgen:"set"`
	MoreKeys    map[K]struct{}   `opt:"MoreKeys" optgen:"set,append"`
	Funcs       map[K]func()     `opt:"Funcs" optgen:"put"`
	Enum        map[K]uint8      `opt:"Enum" optgen:"put"`
	StringValue map[string]uint8 `opt:"StringValue" optgen:"put"`
	FieldSet    optionsFieldSet
}
