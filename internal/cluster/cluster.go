package cluster

import (
	"context"
	"fmt"

	"github.com/go-openapi/strfmt"

	"github.com/filanov/bm-inventory/models"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=cluster.go -package=cluster -destination=mock_cluster_api.go

type StateAPI interface {
	// Refresh state in case of hosts update7
	RefreshStatus(ctx context.Context, c *models.Cluster, db *gorm.DB) (*UpdateReply, error)
}

type RegistrationAPI interface {
	// Register a new cluster
	RegisterCluster(ctx context.Context, c *models.Cluster) error
	//deregister cluster
	DeregisterCluster(ctx context.Context, c *models.Cluster) error
}

type InstallationAPI interface {
	// Install cluster
	Install(ctx context.Context, c *models.Cluster, db *gorm.DB) error
	// Get the cluster master nodes ID's
	GetMasterNodesIds(ctx context.Context, c *models.Cluster, db *gorm.DB) ([]*strfmt.UUID, error)
}

type API interface {
	StateAPI
	RegistrationAPI
	InstallationAPI
}

type Manager struct {
	insufficient    StateAPI
	ready           StateAPI
	installing      StateAPI
	installed       StateAPI
	error           StateAPI
	registrationAPI RegistrationAPI
	installationAPI InstallationAPI
}

func NewManager(log logrus.FieldLogger, db *gorm.DB) *Manager {
	return &Manager{
		insufficient:    NewInsufficientState(log, db),
		ready:           NewReadyState(log, db),
		installing:      NewInstallingState(log, db),
		installed:       NewInstalledState(log, db),
		error:           NewErrorState(log, db),
		registrationAPI: NewRegistrar(log, db),
		installationAPI: NewInstaller(log, db),
	}
}

func (m *Manager) getCurrentState(status string) (StateAPI, error) {
	switch status {
	case "":
	case clusterStatusInsufficient:
		return m.insufficient, nil
	case clusterStatusReady:
		return m.ready, nil
	case clusterStatusInstalling:
		return m.installing, nil
	case clusterStatusInstalled:
		return m.installed, nil
	case clusterStatusError:
		return m.error, nil
	}
	return nil, fmt.Errorf("not supported cluster status: %s", status)
}

func (m *Manager) RegisterCluster(ctx context.Context, c *models.Cluster) error {
	return m.registrationAPI.RegisterCluster(ctx, c)
}

func (m *Manager) DeregisterCluster(ctx context.Context, c *models.Cluster) error {
	return m.registrationAPI.DeregisterCluster(ctx, c)
}

func (m *Manager) RefreshStatus(ctx context.Context, c *models.Cluster, db *gorm.DB) (*UpdateReply, error) {
	state, err := m.getCurrentState(swag.StringValue(c.Status))
	if err != nil {
		return nil, err
	}
	return state.RefreshStatus(ctx, c, db)
}

func (m *Manager) Install(ctx context.Context, c *models.Cluster, db *gorm.DB) error {
	return m.installationAPI.Install(ctx, c, db)
}
func (m *Manager) GetMasterNodesIds(ctx context.Context, c *models.Cluster, db *gorm.DB) ([]*strfmt.UUID, error) {
	return m.installationAPI.GetMasterNodesIds(ctx, c, db)
}
