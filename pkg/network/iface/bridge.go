package iface

import (
	"fmt"
	"syscall"

	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

type Bridge struct {
	*netlink.Bridge
}

func NewBridge(name string) *Bridge {
	vlanFiltering := true
	br := &Bridge{
		Bridge: &netlink.Bridge{
			LinkAttrs:     netlink.LinkAttrs{Name: name},
			VlanFiltering: &vlanFiltering,
		},
	}

	return br
}

// Ensure bridge
// set promiscuous mod default
func (br *Bridge) Ensure() error {
	if err := netlink.LinkAdd(br.Bridge); err != nil && err != syscall.EEXIST {
		return fmt.Errorf("add iface failed, error: %w, iface: %v", err, br)
	}

	// Re-fetch link to read all attributes and if it already existed,
	// ensure it's really a bridge with similar configuration
	tempBr, err := fetchByName(br.Name)
	if err != nil {
		return err
	}

	if tempBr.Promisc != 1 {
		if err := netlink.SetPromiscOn(br); err != nil {
			return fmt.Errorf("set promiscuous mode failed, error: %w, iface: %v", err, br)
		}
	}

	if tempBr.OperState != netlink.OperUp {
		if err := netlink.LinkSetUp(br); err != nil {
			return err
		}
	}

	// TODO ensure vlan filtering

	// Re-fetch bridge to ensure br.Bridge contains all latest attributes.
	br.Bridge, err = fetchByName(br.Name)
	if err != nil {
		return err
	}

	return nil
}

func (br *Bridge) Delete() error {
	if err := netlink.LinkDel(br); err != nil {
		return fmt.Errorf("could not delete link %s, error: %w", br.Name, err)
	}

	return nil
}

func fetchByName(name string) (*netlink.Bridge, error) {
	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("could not lookup link %s, error: %w", name, err)
	}

	br, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("%s already exists but is not a iface", name)
	}

	return br, nil
}

// find the relative complement of A (left side) in B (right side)
func relativeComplement(A, B []netlink.Addr) []netlink.Addr {
	var complement []netlink.Addr
	for i := range B  {
		for j := range A {
			if B[i].Equal(A[j]) {
				break
			}
		}
		complement = append(complement, B[i])
	}

	return complement
}

func (br *Bridge) configIPv4AddrFromSlave(slave *Link) error {
	slaveAddrList, err := netlink.AddrList(slave, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("list IPv4 address of %s failed, error: %w", slave.Attrs().Name, err)
	}
	brAddrList, err := netlink.AddrList(br.Bridge, netlink.FAMILY_V4)
	if err != nil {
		return fmt.Errorf("list IPv4 address of %s failed, error: %w", br.Name, err)
	}
	addList := relativeComplement(brAddrList, slaveAddrList)
	delList := relativeComplement(slaveAddrList, brAddrList)

	for _, addr := range addList {
		if err := netlink.AddrReplace(br.Bridge, &addr); err != nil {
			return fmt.Errorf("could not add address, error: %w, link: %s, addr: %+v", err, br.Name, addr)
		}
	}
	for _, addr := range delList {
		if err := netlink.AddrDel(br.Bridge, &addr); err != nil {
			return fmt.Errorf("could not add address, error: %w, link: %s, addr: %+v", err, br.Name, addr)
		}
	}

	return nil
}

func (br *Bridge) replaceRoutes(slave *Link) error {
	brLink := &Link{Link: br}
	if err := brLink.ReplaceRoutes(slave); err != nil {
		return fmt.Errorf("replaces routes from %s to %s failed, error: %w", br.Name, slave.Attrs().Name, err)
	}

	return nil
}

func (br *Bridge) ConfigIPv4AddrFromSlave(slave *Link, routes []netlink.Route) error {
	if err := br.configIPv4AddrFromSlave(slave); err != nil {
		return fmt.Errorf("configure IPv4 addresses from slave link %s failed, error: %w", slave.Attrs().Name, err)
	}

	if err := br.replaceRoutes(slave); err != nil {
		return fmt.Errorf("replace route rules of slave link %s failed, error: %w", slave.Attrs().Name, err)
	}

	// configure route rules passed by parameters
	for i := range routes {
		if err := netlink.RouteReplace(&routes[i]); err != nil {
			klog.Errorf("could not replace route, error: %s, route: %v", err.Error(), routes[i])
		}
	}

	return nil
}

func (br *Bridge) delAddr() error {
	addrList, err := br.ListAddr()
	if err != nil {
		return fmt.Errorf("list IPv4 address of %s failed, error: %w", br.Name, err)
	}

	num := len(addrList)
	if num == 0 {
		return nil
	}
	if num > 1 {
		return fmt.Errorf("not support multiple addresses, iface: %s, address number: %d", br.Name, num)
	}

	if err := netlink.AddrDel(br, &addrList[0]); err != nil {
		return fmt.Errorf("delete address of %s failed, error: %w", br.Name, err)
	}

	return nil
}

func (br *Bridge) ListAddr() ([]netlink.Addr, error) {
	return netlink.AddrList(br, netlink.FAMILY_V4)
}
