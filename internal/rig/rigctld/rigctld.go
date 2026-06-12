package rigctld

import (
	"context"

	"github.com/szporwolik/cqops/internal/rig"
)

type Rigctld struct {
	Host      string
	Port      int
	TimeoutMS int
}

func New(host string, port int, timeoutMS int) *Rigctld {
	return &Rigctld{
		Host:      host,
		Port:      port,
		TimeoutMS: timeoutMS,
	}
}

func (r *Rigctld) Status(ctx context.Context) (rig.RigStatus, error) {
	return rig.RigStatus{
		Provider:  "rigctld",
		Connected: false,
	}, nil
}
