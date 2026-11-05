package stages45

import "time"

type optionsFieldSet uint64

type FailMode string

const (
	FailOpen   FailMode = "open"
	FailClosed FailMode = "closed"
)

type RetryOption func()

func makeRetry(time.Duration) RetryOption { return func() {} }

type options[T any] struct {
	enabled    bool                `opt:"Enabled" optgen:"resets=pointer|callback|payload"`
	filter     func(string) bool   `opt:"Filter" optgen:"enables=filterOn"`
	filterOn   bool                `opt:"-"`
	timeout    time.Duration       `opt:"Timeout" optgen:"enables=durationOn" optval:"nonpositive" optcheck:"min=1s"`
	durationOn bool                `opt:"-"`
	retry      RetryOption         `optfn:"Retry=makeRetry" optgen:"enables=retryOn"`
	retryOn    bool                `opt:"-"`
	mode       FailMode            `opt:"Mode" optcheck:"oneof=[FailOpen,FailClosed]"`
	limited    int                 `opt:"Limited" optgen:"onfail=skip" optcheck:"min=1"`
	plain      int                 `opt:"Plain" optcheck:"min=1"`
	slice      []string            `opt:"Slice" optgen:"enables=sliceOn"`
	sliceOn    bool                `opt:"-"`
	appended   []int               `opt:"Appended" optgen:"append,enables=appendOn"`
	appendOn   bool                `opt:"-"`
	keys       map[string]struct{} `opt:"Keys" optgen:"set,enables=keysOn"`
	keysOn     bool                `opt:"-"`
	entries    map[string]int      `opt:"Entries" optgen:"put,enables=entriesOn"`
	entriesOn  bool                `opt:"-"`
	pointer    *int                `opt:"-"`
	callback   func()              `opt:"-"`
	payload    T                   `opt:"-"`
	set        optionsFieldSet
}
