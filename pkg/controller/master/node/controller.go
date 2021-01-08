package node

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1alpha1 "github.com/rancher/harvester-network-controller/pkg/apis/network.harvester.cattle.io/v1alpha1"
	"github.com/rancher/harvester-network-controller/pkg/config"
	ctlnetworkv1alpha1 "github.com/rancher/harvester-network-controller/pkg/generated/controllers/network.harvester.cattle.io/v1alpha1"
)

const (
	controllerName = "node-controller"
)

type Handler struct {
	hostNetworkClient ctlnetworkv1alpha1.HostNetworkClient
	hostNetworkCache  ctlnetworkv1alpha1.HostNetworkCache
}

func Register(ctx context.Context, management *config.Management) error {
	nodes := management.CoreFactory.Core().V1().Node()
	hostNetworks := management.HarvesterNetworkFactory.Network().V1alpha1().HostNetwork()

	handler := &Handler{
		hostNetworkClient: hostNetworks,
		hostNetworkCache:  hostNetworks.Cache(),
	}

	nodes.OnChange(ctx, controllerName, handler.OnChange)

	return nil
}

func NewHostFromNode(node *corev1.Node) *networkv1alpha1.HostNetwork {
	return &networkv1alpha1.HostNetwork{
		ObjectMeta: metav1.ObjectMeta{
			Name: node.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "v1",
					Kind:       "Node",
					Name:       node.Name,
					UID:        node.UID,
				},
			},
		},
	}
}

func (h Handler) OnChange(key string, node *corev1.Node) (*corev1.Node, error) {
	if node == nil || node.DeletionTimestamp != nil {
		return nil, nil
	}

	_, err := h.hostNetworkCache.Get(node.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if _, err := h.hostNetworkClient.Create(NewHostFromNode(node)); err != nil {
				return nil, err
			}
			return node, nil
		}
		return nil, err
	}

	return node, nil
}
