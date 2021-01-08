package network

import "github.com/vishvananda/netlink"

type IsolatedNetwork interface {
	Setup(nic string, conf NetworkConfig) error
	Repeal() error
	AddLocalArea(id int) error
	RemoveLocalArea(id int) error
	Monitor() error
}

type NetworkConfig struct {
	routes []netlink.Route
}
