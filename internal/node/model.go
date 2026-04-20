package node

import "github.com/google/uuid"

type Protocol string

const (
	ProtoVMess     Protocol = "vmess"
	ProtoVLESS     Protocol = "vless"
	ProtoTrojan    Protocol = "trojan"
	ProtoSS        Protocol = "ss"
	ProtoTUIC      Protocol = "tuic"
	ProtoHysteria2 Protocol = "hysteria2"
)

type Node struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol Protocol `json:"protocol"`
	Address  string   `json:"address"`
	Port     int      `json:"port"`

	// Auth
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
	Method   string `json:"method,omitempty"`

	// VMess
	AlterID  int    `json:"alter_id,omitempty"`
	Security string `json:"security,omitempty"`

	// VLESS / Trojan
	Flow       string `json:"flow,omitempty"`
	Encryption string `json:"encryption,omitempty"`

	// Transport
	Network  string `json:"network,omitempty"`
	Path     string `json:"path,omitempty"`
	Host     string `json:"host,omitempty"`
	GrpcSvc  string `json:"grpc_svc,omitempty"`
	GrpcMode string `json:"grpc_mode,omitempty"`

	// TLS
	TLS         string `json:"tls,omitempty"`
	SNI         string `json:"sni,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	ALPN        string `json:"alpn,omitempty"`
	Insecure    bool   `json:"insecure,omitempty"`

	// Reality
	PublicKey string `json:"public_key,omitempty"`
	ShortID   string `json:"short_id,omitempty"`
	SpiderX   string `json:"spider_x,omitempty"`

	// TUIC
	CongestionControl string `json:"congestion_control,omitempty"`

	// Hysteria2
	ObfsType     string `json:"obfs_type,omitempty"`
	ObfsPassword string `json:"obfs_password,omitempty"`
	Ports        string `json:"ports,omitempty"`
	PinSHA256    string `json:"pin_sha256,omitempty"`
}

func NewID() string {
	return uuid.New().String()
}
