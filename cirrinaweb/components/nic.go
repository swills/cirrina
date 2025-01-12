package components

type NIC struct {
	ID          string
	Name        string
	NameOrID    string
	Description string
	VM          VM
	Uplink      string
}
