package nad

import (
	"context"

	nadv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/rancher/harvester-network-controller/pkg/config"
)

const (
	controllerName = "nad-controller"
)

type Handler struct {
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

	return nad, nil
}

func (h Handler) OnRemove(key string, nad *nadv1.NetworkAttachmentDefinition) (*nadv1.NetworkAttachmentDefinition, error) {
	if nad == nil {
		return nil, nil
	}

	if nad.Spec.Config == "" {
		return nad, nil
	}

	return nad, nil
}
