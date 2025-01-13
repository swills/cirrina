package components

type NIC struct {
	ID          string
	Name        string
	NameOrID    string
	Description string
	VM          VM
	Uplink      Switch
	Type        string
	DevType     string
	RateLimited bool
	RateIn      string
	RateOut     string
}
