package comm

type JsonData struct {
	JsonData []Conn `json:"conn"`
}

type Conn struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	CertType int    `json:"cert_type"`
	Cert     string `json:"cert"`
}

const (
	CertAgent    = 0 // ssh-agent (SSH_AUTH_SOCK)
	CertPassword = 1 // password; empty Cert -> prompt at connect
	CertKeyFile  = 2 // private key file; passphrase is prompted if needed

	DefaultPort = 22
)
