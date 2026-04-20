package builder

import (
	"fmt"
	"math/rand"
	"net"
)

// Ports holds the randomly assigned listen ports for one session.
type Ports struct {
	DNS      int `json:"dns"`
	Mixed    int `json:"mixed"`
	Redirect int `json:"redirect"`
	TProxy   int `json:"tproxy"`
}

// RandomPorts picks 4 available TCP ports in the range 10000–59999.
func RandomPorts() Ports {
	return Ports{
		DNS:      freePort(),
		Mixed:    freePort(),
		Redirect: freePort(),
		TProxy:   freePort(),
	}
}

func freePort() int {
	for {
		p := 10000 + rand.Intn(50000)
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err != nil {
			continue
		}
		l.Close()
		return p
	}
}
