package value

import (
	"reflect"

	"github.com/grafana/agent/pkg/river/vm/internal/rivertags"
)

// tagsCache caches the river tags for a struct type. This is never cleared,
// but since most structs will be statically created throughout the lifetime
// of the process, this will consume a negligable amount of memory.
var tagsCache = make(map[reflect.Type]rivertags.Fields)

func getCachedTags(t reflect.Type) rivertags.Fields {
	if t.Kind() != reflect.Struct {
		panic("getCachedTags called with non-struct type")
	}

	if entry, ok := tagsCache[t]; ok {
		return entry
	}

	ff := rivertags.Get(t)
	tagsCache[t] = ff
	return ff
}
