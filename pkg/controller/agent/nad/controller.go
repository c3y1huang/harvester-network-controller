package nad

import (
	"context"
	"fmt"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	ctlharv1 "github.com/rancher/harvester/pkg/generated/controllers/harvester.cattle.io/v1alpha1"
	"k8s.io/klog"

	"github.com/rancher/harvester-network-controller/pkg/bridge"
	"github.com/rancher/harvester-network-controller/pkg/config"
	"github.com/rancher/harvester-network-controller/pkg/util"
)

const (
	controllerName = "nad-controller"
)

type Handler struct {
	settingCache ctlharv1.SettingCache
}

func Register(ctx context.Context, management *config.Management) error {
	nad := management.CniFactory.K8s().V1().NetworkAttachmentDefinition()

	handler := &Handler{}

	nad.OnChange(ctx, controllerName, handler.OnChange)
	nad.OnRemove(ctx, controllerName, handler.OnRemove)

	return nil
}

func (h Handler) OnChange(key string, nad *nadv1.NetworkAttachmentDefinition) (*nadv1.NetworkAttachmentDefinition, error) {
	if nad == nil || nad.DeletionTimestamp != nil {
		return nil, nil
	}

	if nad.Spec.Config == "" {
		return nad, nil
	}

	klog.Infof("nad configuration %s has been changed: %s", nad.Name, nad.Spec.Config)
	netconf, err := util.DecodeNetConf(nad.Spec.Config)
	if err != nil {
		return nil, err
	}

	l, err := h.getLink()
	if err != nil {
		return nil, err
	}

	// TODO delete previous vlan id when update nad

	if l != nil {
		if err := l.AddBridgeVlan(uint16(netconf.Vlan)); err != nil {
			return nil, err
		}
	}

	return nad, nil
}

func (h Handler) OnRemove(key string, nad *nadv1.NetworkAttachmentDefinition) (*nadv1.NetworkAttachmentDefinition, error) {
	if nad == nil {
		return nil, nil
	}

	if nad.Spec.Config == "" {
		return nad, nil
	}

	klog.Infof("nad configuration %s has been deleted.", nad.Name)
	netconf, err := util.DecodeNetConf(nad.Spec.Config)
	if err != nil {
		return nil, err
	}

	l, err := h.getLink()
	if err != nil {
		return nil, err
	}

	if l != nil {
		klog.Infof("remove nad %s with vid %d from the list", key, netconf.Vlan)
		if err := l.DelBridgeVlan(uint16(netconf.Vlan)); err != nil {
			return nil, err
		}
	}

	return nad, nil
}

func (h Handler) getLink() (*bridge.Link, error) {
	setting, err := h.settingCache.Get(util.NetworkSettingName)
	if err != nil {
		return nil, fmt.Errorf("get vlan setting failed, error: %+v", err)
	}

	networkSetting, err := util.DecodeNetworkSettings(setting.Value)
	if err != nil {
		return nil, err
	}

	return bridge.GetLink(networkSetting.NIC)
}
