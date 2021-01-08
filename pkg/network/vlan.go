package network

import (
	"fmt"

	"github.com/vishvananda/netlink"

	"github.com/rancher/harvester-network-controller/pkg/network/iface"
)

type Vlan struct {
	bridge *iface.Bridge
	nic    *iface.Link
}

func NewVlan(bridge string) *Vlan {
	br := iface.NewBridge(bridge)
	return &Vlan{bridge: br}
}

func NewVlanWithNic(bridge, nic string) (*Vlan, error) {
	vlan := NewVlan(bridge)
	l, err := iface.GetLink(nic)
	if err != nil {
		return nil, err
	}
	vlan.nic = l

	return vlan, nil
}

func (v *Vlan) Monitor() {}

func (v *Vlan) Setup(nic string, conf NetworkConfig) error {
	// ensure bridge
	if err := v.bridge.Ensure(); err != nil {
		return fmt.Errorf("ensure bridge %s failed, error: %w", v.bridge.Name, err)
	}

	l, err := iface.GetLink(nic)
	if err != nil {
		return err
	}

	// setup L2 layer network
	if err := v.setupL2(l); err != nil {
		return err
	}
	// setup L3 layer network
	if err := v.setupL3(l, conf.routes); err != nil {
		return err
	}

	v.nic = l
	return nil
}

func (v *Vlan) setupL2(nic *iface.Link) error {
	return nic.SetMaster(v.bridge)
}

func (v *Vlan) setupL3(nic *iface.Link, routes []netlink.Route) error {
	// set ebtables rules
	if err := nic.SetRules4DHCP(); err != nil {
		return err
	}
	// configure IPv4 address
	return v.bridge.ConfigIPv4AddrFromSlave(nic, routes)
}

func (v *Vlan) unsetL3() error {
	if err := v.nic.UnsetRules4DHCP(); err != nil {
		return err
	}

	if err := v.nic.ReplaceRoutes(&iface.Link{v.bridge}); err != nil {
		return fmt.Errorf("replace routes from %s to %s failed, error: %w", v.nic.Attrs().Name, v.bridge.Name, err)
	}

	return nil
}

func (v *Vlan) AddLocalArea(id int) error {
	if v.nic == nil {
		return fmt.Errorf("physical nic vlan network")
	}
	return v.nic.AddBridgeVlan(uint16(id))
}

func (v *Vlan) RemoveLocalArea(id int) error {
	if v.nic == nil {
		return fmt.Errorf("physical nic vlan network")
	}
	return v.nic.DelBridgeVlan(uint16(id))
}

func (v *Vlan) Repeal() error {
	if v.nic == nil {
		return fmt.Errorf("vlan network haven't attached a NIC")
	}

	if err := v.unsetL3(); err != nil {
		return err
	}

	return v.bridge.Delete()
}
