package internals

const (
	F0 uint8 = 1 << iota
	F1
	F2
	F3
	F4
	F5
	F6
	F7
)

type Bitmap struct {
	data uint8
}

func NewBitmap(data uint8) *Bitmap {
	return &Bitmap{
		data: data,
	}
}

func (m *Bitmap) Set(flag uint8) uint8 {
	m.data = m.data | flag
	return m.data
}

func (m *Bitmap) Clear(flag uint8) uint8 {
	m.data = m.data &^ flag
	return m.data
}

func (m *Bitmap) Toggle(flag uint8) uint8 {
	m.data = m.data ^ flag
	return m.data
}
func (m *Bitmap) Has(flag uint8) bool {
	return m.data&flag != 0
}

func (m *Bitmap) Value() uint8 {
	return m.data
}
