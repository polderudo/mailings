package internals

type ReIter[T any] struct {
	index int
	d     []*T
}

func NewReIterer[T any](data []*T) *ReIter[T] {
	r := &ReIter[T]{
		index: 0,
		d:     data,
	}
	return r
}

func (m *ReIter[T]) Next() *T {
	if m.index == len(m.d) {
		m.index = 0
	}
	ret := m.d[m.index]
	m.index++
	return ret
}
