package comm

type JsonData struct {
	JsonData []Conn `json:"conn"`
}

type Conn struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Cert_Type int    `json:"cert_type"`
	Cert      string `json:"cert"`
}

const (
	TuiMain int = iota
	TuiAddItem int = iota
	TuiDeleteItem int = iota
)