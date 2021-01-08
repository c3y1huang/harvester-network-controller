package config

import (
	"context"

	cniv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	ctlcni "github.com/rancher/harvester/pkg/generated/controllers/k8s.cni.cncf.io"
	"github.com/rancher/lasso/pkg/controller"
	ctlcore "github.com/rancher/wrangler-api/pkg/generated/controllers/core"
	wcrd "github.com/rancher/wrangler/pkg/crd"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/schemes"
	"github.com/rancher/wrangler/pkg/start"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/rest"

	networkv1alpha1 "github.com/rancher/harvester-network-controller/pkg/apis/network.harvester.cattle.io/v1alpha1"
	ctlnetwork "github.com/rancher/harvester-network-controller/pkg/generated/controllers/network.harvester.cattle.io"
	"github.com/rancher/harvester-network-controller/pkg/util/crd"
)

var (
	localSchemeBuilder = runtime.SchemeBuilder{
		networkv1alpha1.AddToScheme,
	}
	AddToScheme = localSchemeBuilder.AddToScheme
	Scheme      = runtime.NewScheme()
)

func init() {
	utilruntime.Must(AddToScheme(Scheme))
	utilruntime.Must(schemes.AddToScheme(Scheme))
}

type RegisterFunc func(context.Context, *Management) error

type Management struct {
	ctx               context.Context
	ControllerFactory controller.SharedControllerFactory

	HarvesterNetworkFactory *ctlnetwork.Factory
	CniFactory              *ctlcni.Factory
	CoreFactory             *ctlcore.Factory

	starters []start.Starter
}

func (s *Management) Start(threadiness int) error {
	return start.All(s.ctx, threadiness, s.starters...)
}

func (s *Management) Register(ctx context.Context, config *rest.Config, registerFuncList []RegisterFunc) error {
	if err := createCRDsIfNotExisted(ctx, config); err != nil {
		return err
	}

	for _, f := range registerFuncList {
		if err := f(ctx, s); err != nil {
			return err
		}
	}

	return nil
}

func createCRDsIfNotExisted(ctx context.Context, config *rest.Config) error {
	factory, err := crd.NewFactoryFromClient(ctx, config)
	if err != nil {
		return err
	}
	return factory.
		CreateCRDsIfNotExisted(
			crd.NonNamespacedFromGV(networkv1alpha1.SchemeGroupVersion, "HostNetwork"),
		).
		CreateCRDsIfNotExisted(
			createNetworkAttachmentDefinitionCRD(),
		).
		Wait()
}

func createNetworkAttachmentDefinitionCRD() wcrd.CRD {
	nad := crd.FromGV(cniv1.SchemeGroupVersion, "NetworkAttachmentDefinition")
	nad.PluralName = "network-attachment-definitions"
	nad.SingularName = "network-attachment-definition"
	return nad
}

func SetupManagement(ctx context.Context, restConfig *rest.Config) (*Management, error) {
	factory, err := controller.NewSharedControllerFactoryFromConfig(restConfig, Scheme)
	if err != nil {
		return nil, err
	}

	opts := &generic.FactoryOptions{
		SharedControllerFactory: factory,
	}

	management := &Management{
		ctx: ctx,
	}

	harvesterNetwork, err := ctlnetwork.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	management.HarvesterNetworkFactory = harvesterNetwork
	management.starters = append(management.starters, harvesterNetwork)

	core, err := ctlcore.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	management.CoreFactory = core
	management.starters = append(management.starters, core)

	cni, err := ctlcni.NewFactoryFromConfigWithOptions(restConfig, opts)
	if err != nil {
		return nil, err
	}
	management.CniFactory = cni
	management.starters = append(management.starters, cni)

	return management, nil
}
