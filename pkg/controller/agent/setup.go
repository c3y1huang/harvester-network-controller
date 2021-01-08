package daemonset

import (
	"github.com/rancher/harvester-network-controller/pkg/config"
	"github.com/rancher/harvester-network-controller/pkg/controller/daemonset/hostnetwork"
	"github.com/rancher/harvester-network-controller/pkg/controller/daemonset/nad"
)

var RegisterFuncList = []config.RegisterFunc{
	hostnetwork.Register,
	nad.Register,
}
