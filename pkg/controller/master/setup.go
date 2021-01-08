package daemonset

import (
	"github.com/rancher/harvester-network-controller/pkg/config"
	"github.com/rancher/harvester-network-controller/pkg/controller/master/node"
)

var RegisterFuncList = []config.RegisterFunc{
	node.Register,
}
