package hostnetwork

import (
	"context"

	networkv1alpha1 "github.com/rancher/harvester-network-controller/pkg/apis/network.harvester.cattle.io/v1alpha1"

	"github.com/rancher/harvester-network-controller/pkg/config"
)

const (
	controllerName = "host-controller"
)

type Handler struct {
}

func Register(ctx context.Context, management *config.Management) error {
	hosts := management.HarvesterNetworkFactory.Network().V1alpha1().HostNetwork()

	handler := &Handler{}
	hosts.OnChange(ctx, controllerName, handler.OnChange)
	hosts.OnRemove(ctx, controllerName, handler.OnRemove)

	return nil
}

func (h Handler) OnChange(key string, host *networkv1alpha1.HostNetwork) (*networkv1alpha1.HostNetwork, error) {
	if host == nil || host.DeletionTimestamp != nil {
		return nil, nil
	}

	return host, nil
}

func (h Handler) OnRemove(key string, host *networkv1alpha1.HostNetwork) (*networkv1alpha1.HostNetwork, error) {
	if host == nil {
		return nil, nil
	}

	return host, nil
}
