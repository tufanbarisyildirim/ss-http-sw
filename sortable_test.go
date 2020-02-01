package ss_http_sw

import (
	"reflect"
	"sort"
	"testing"
)

func TestSortableInt64_Gen(t *testing.T) {
	list := []int64{99, 10, 11, 33, 54}
	sort.Sort(SortableInt64(list))
	if !reflect.DeepEqual(list, []int64{10, 11, 33, 54, 99}) {
		t.Fail()
	}
}