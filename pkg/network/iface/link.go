package iface

import (
	"fmt"

	"github.com/vishvananda/netlink"
	"k8s.io/klog"
	"k8s.io/utils/exec"
	"k8s.io/utils/net/ebtables"
)

const defaultPVID = uint16(1)

type Link struct {
	netlink.Link
}

// GetLink by name
func GetLink(name string) (*Link, error) {
	if name == "" {
		return nil, fmt.Errorf("link name could not be empty string")
	}

	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup link, error: %w, link: %s", err, name)
	}

	return &Link{Link: l}, nil
}

// AddBridgeVlan adds a new vlan filter entry
// Equivalent to: `bridge vlan add dev DEV vid VID master`
func (l *Link) AddBridgeVlan(vid uint16) error {
	if vid == defaultPVID {
		return nil
	}

	if err := netlink.BridgeVlanAdd(l.Link, vid, false, false, false, true); err != nil {
		return fmt.Errorf("add iface vlan failed, error: %v, link: %s, vid: %d", err, l.Attrs().Name, vid)
	}

	return nil
}

// DelBridgeVlan adds a new vlan filter entry
// Equivalent to: `bridge vlan del dev DEV vid VID master`
func (l *Link) DelBridgeVlan(vid uint16) error {
	if vid == defaultPVID {
		return nil
	}

	if err := netlink.BridgeVlanDel(l.Link, vid, false, false, false, true); err != nil {
		return fmt.Errorf("delete iface vlan failed, error: %v, link: %s, vid: %d", err, l.Attrs().Name, vid)
	}

	return nil
}

func (l *Link) SetMaster(br *Bridge) error {
	if l.Attrs().MasterIndex == br.Index {
		return nil
	}

	return netlink.LinkSetMaster(l, br)
}

func (l *Link) SetNoMaster() error {
	return netlink.LinkSetNoMaster(l)
}

// allow to receive DHCP packages after attaching with bridge
func (l *Link) SetRules4DHCP() error {
	exec := exec.New()
	runner := ebtables.New(exec)
	var ruleArgs []string

	ruleArgs = append(ruleArgs, "-p", "IPv4", "-d", l.Attrs().HardwareAddr.String(), "-i", l.Attrs().Name,
		"--ip-proto", "udp", "--ip-dport", "67:68", "-j", "DROP")
	_, err := runner.EnsureRule(ebtables.Append, ebtables.TableBroute, ebtables.ChainBrouting, ruleArgs...)
	if err != nil {
		return fmt.Errorf("set ebtables rules failed, error: %w", err)
	}

	return nil
}

func (l *Link) UnsetRules4DHCP() error {
	exec := exec.New()
	runner := ebtables.New(exec)
	var ruleArgs []string

	ruleArgs = append(ruleArgs, "-p", "IPv4", "-d", l.Attrs().HardwareAddr.String(), "-i", l.Attrs().Name,
		"--ip-proto", "udp", "--ip-dport", "67:68", "-j", "DROP")
	if err := runner.DeleteRule(ebtables.TableBroute, ebtables.ChainBrouting, ruleArgs...); err != nil {
		return fmt.Errorf("delete ebtables rules failed, error: %w", err)
	}

	return nil
}

func (l *Link) ReplaceRoutes(replaced *Link) error {
	routeList, err := netlink.RouteList(replaced, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("could not list routes, error: %w, link: %s", err, replaced.Attrs().Name)
	}
	for _, route := range routeList {
		route.LinkIndex = l.Attrs().Index
		if err := netlink.RouteReplace(&route); err != nil {
			klog.Infof("could not replace route %v and it will be removed", route)
			route.LinkIndex = replaced.Attrs().Index
			// remove route rule that can not be replaced, such as
			// the auto-generated route `172.16.0.0/16 dev harvester-br0 proto kernel scope link src 172.16.0.76`
			if err := netlink.RouteDel(&route); err != nil {
				klog.Errorf("could not delete route, error: %s, route: %v", err.Error(), route)
			}
		} else {
			klog.Infof("replace route, %+v", route)
		}
	}
	return nil
}