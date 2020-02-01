package ss_http_sw

type SortableInt64 []int64

func (p SortableInt64) Len() int           { return len(p) }
func (p SortableInt64) Less(i, j int) bool { return p[i] < p[j] }
func (p SortableInt64) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
