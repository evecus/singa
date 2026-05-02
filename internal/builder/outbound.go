package builder

import (
	"fmt"
	"strings"

	"github.com/singa/internal/node"
)

// NodeToOutbound converts a parsed Node to a sing-box outbound map.
func NodeToOutbound(n *node.Node, tag string) (map[string]interface{}, error) {
	switch n.Protocol {
	case node.ProtoVMess:
		return vmessOB(n, tag)
	case node.ProtoVLESS:
		return vlessOB(n, tag)
	case node.ProtoTrojan:
		return trojanOB(n, tag)
	case node.ProtoSS:
		return ssOB(n, tag)
	case node.ProtoTUIC:
		return tuicOB(n, tag)
	case node.ProtoHysteria2:
		return hy2OB(n, tag)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", n.Protocol)
	}
}

func vmessOB(n *node.Node, tag string) (map[string]interface{}, error) {
	ob := M{
		"type":        "vmess",
		"tag":         tag,
		"server":      n.Address,
		"server_port": n.Port,
		"uuid":        n.UUID,
		"alter_id":    n.AlterID,
		"security":    nonEmpty(n.Security, "auto"),
	}
	setTransport(ob, n)
	setTLS(ob, n)
	return ob, nil
}

func vlessOB(n *node.Node, tag string) (map[string]interface{}, error) {
	ob := M{
		"type":        "vless",
		"tag":         tag,
		"server":      n.Address,
		"server_port": n.Port,
		"uuid":        n.UUID,
	}
	if n.Flow != "" {
		ob["flow"] = n.Flow
	}
	setTransport(ob, n)
	setTLS(ob, n)
	return ob, nil
}

func trojanOB(n *node.Node, tag string) (map[string]interface{}, error) {
	ob := M{
		"type":        "trojan",
		"tag":         tag,
		"server":      n.Address,
		"server_port": n.Port,
		"password":    n.Password,
	}
	if n.Flow != "" {
		ob["flow"] = n.Flow
	}
	setTransport(ob, n)
	setTLS(ob, n)
	return ob, nil
}

func ssOB(n *node.Node, tag string) (map[string]interface{}, error) {
	return M{
		"type":        "shadowsocks",
		"tag":         tag,
		"server":      n.Address,
		"server_port": n.Port,
		"method":      n.Method,
		"password":    n.Password,
	}, nil
}

func tuicOB(n *node.Node, tag string) (map[string]interface{}, error) {
	ob := M{
		"type":                "tuic",
		"tag":                 tag,
		"server":              n.Address,
		"server_port":         n.Port,
		"uuid":                n.UUID,
		"password":            n.Password,
		"congestion_control":  nonEmpty(n.CongestionControl, "bbr"),
	}
	setTLS(ob, n)
	return ob, nil
}

func hy2OB(n *node.Node, tag string) (map[string]interface{}, error) {
	ob := M{
		"type":     "hysteria2",
		"tag":      tag,
		"server":   n.Address,
		"password": n.Password,
	}
	if n.Ports != "" {
		ob["server_port"] = n.Ports
	} else {
		ob["server_port"] = n.Port
	}
	if n.ObfsType != "" {
		ob["obfs"] = M{
			"type":     n.ObfsType,
			"password": n.ObfsPassword,
		}
	}
	setTLS(ob, n)
	return ob, nil
}

// ── TLS ────────────────────────────────────────────────────────────────────

func setTLS(ob M, n *node.Node) {
	if n.TLS == "" {
		return
	}
	tls := M{"enabled": true}
	if n.SNI != "" {
		tls["server_name"] = n.SNI
	}
	if n.Insecure {
		tls["insecure"] = true
	}
	if n.Fingerprint != "" {
		tls["utls"] = M{"enabled": true, "fingerprint": n.Fingerprint}
	}
	if n.ALPN != "" {
		tls["alpn"] = strings.Split(n.ALPN, ",")
	}
	if n.TLS == "reality" {
		reality := M{"enabled": true, "public_key": n.PublicKey}
		if n.ShortID != "" {
			reality["short_id"] = n.ShortID
		}
		tls["reality"] = reality
		delete(tls, "insecure")
	}
	ob["tls"] = tls
}

// ── Transport ──────────────────────────────────────────────────────────────

func setTransport(ob M, n *node.Node) {
	t := transportObj(n)
	if t != nil {
		ob["transport"] = t
	}
}

func transportObj(n *node.Node) M {
	switch n.Network {
	case "ws":
		t := M{"type": "ws"}
		if n.Path != "" {
			t["path"] = n.Path
		}
		if n.Host != "" {
			t["headers"] = M{"Host": n.Host}
		}
		return t
	case "grpc":
		t := M{"type": "grpc"}
		if n.GrpcSvc != "" {
			t["service_name"] = n.GrpcSvc
		}
		return t
	case "http":
		t := M{"type": "http"}
		if n.Host != "" {
			t["host"] = []string{n.Host}
		}
		if n.Path != "" {
			t["path"] = n.Path
		}
		return t
	case "httpupgrade":
		t := M{"type": "httpupgrade"}
		if n.Host != "" {
			t["host"] = n.Host
		}
		if n.Path != "" {
			t["path"] = n.Path
		}
		return t
	case "xhttp":
		t := M{"type": "splithttp"}
		if n.Host != "" {
			t["host"] = n.Host
		}
		if n.Path != "" {
			t["path"] = n.Path
		}
		return t
	default:
		return nil
	}
}

// ── Helpers ────────────────────────────────────────────────────────────────

type M = map[string]interface{}

func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
