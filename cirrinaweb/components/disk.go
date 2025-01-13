package components

type Disk struct {
	ID          string
	Name        string
	NameOrID    string
	Description string
	Size        string
	Usage       string
	VM          VM
	Type        string
	DevType     string
	Cache       bool
	Direct      bool
}
