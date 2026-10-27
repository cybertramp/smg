package comm

// JsonData is the root pod: top-level connections plus nested pods (directories).
// Files written before pods existed have no "pods" key and load unchanged.
type JsonData struct {
	JsonData []Conn `json:"conn"`
	Pods     []Pod  `json:"pods,omitempty"`
}

// Pod groups connections and child pods.
type Pod struct {
	Name string `json:"name"`
	Conn []Conn `json:"conn,omitempty"`
	Pods []Pod  `json:"pods,omitempty"`
}

type Conn struct {
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	CertType int    `json:"cert_type"`
	Cert     string `json:"cert"`
	LastUsed int64  `json:"last_used,omitempty"` // unix seconds of last connect
}

const (
	CertAgent    = 0 // ssh-agent (SSH_AUTH_SOCK)
	CertPassword = 1 // password; empty Cert -> prompt at connect
	CertKeyFile  = 2 // private key file; passphrase is prompted if needed

	DefaultPort = 22
)
