package components

type Display struct {
	Enabled        bool
	VNCPort        string
	Width          uint32
	Height         uint32
	VNCWait        bool
	TabletMode     bool
	KeyboardLayout string
}
