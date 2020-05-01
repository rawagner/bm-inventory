package cluster

import (
	"context"
	"fmt"

	"github.com/filanov/bm-inventory/models"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -source=statemachine.go -package=cluster -destination=mock_cluster_api.go
type API interface {
	// Register a new cluster
	RegisterCluster(ctx context.Context, c *models.Cluster) (*UpdateReply, error)
	// Refresh state in case of hosts update7
	RefreshStatus(ctx context.Context, c *models.Cluster, db *gorm.DB) (*UpdateReply, error)
	// Install cluster
	Install(ctx context.Context, c *models.Cluster) (*UpdateReply, error)
	//deregister cluster
	DeregisterCluster(ctx context.Context, c *models.Cluster) (*UpdateReply, error)
}

type State struct {
	insufficient API
	ready        API
	installing   API
	installed    API
	error        API
}

func NewState(log logrus.FieldLogger, db *gorm.DB) *State {
	return &State{
		insufficient: NewInsufficientState(log, db),
		ready:        NewReadyState(log, db),
		installing:   NewInstallingState(log, db),
		installed:    NewInstalledState(log, db),
		error:        NewErrorState(log, db),
	}
}

func (s *State) getCurrentState(status string) (API, error) {
	switch status {
	case "":
	case clusterStatusInsufficient:
		return s.insufficient, nil
	case clusterStatusReady:
		return s.ready, nil
	case clusterStatusInstalling:
		return s.installing, nil
	case clusterStatusInstalled:
		return s.installed, nil
	case clusterStatusError:
		return s.error, nil
	}
	return nil, fmt.Errorf("not supported cluster status: %s", status)
}

func (s *State) RegisterCluster(ctx context.Context, c *models.Cluster) (*UpdateReply, error) {
	state, err := s.getCurrentState(swag.StringValue(c.Status))
	if err != nil {
		return nil, err
	}
	return state.RegisterCluster(ctx, c)
}

func (s *State) RefreshStatus(ctx context.Context, c *models.Cluster, db *gorm.DB) (*UpdateReply, error) {
	state, err := s.getCurrentState(swag.StringValue(c.Status))
	if err != nil {
		return nil, err
	}
	return state.RefreshStatus(ctx, c, db)
}

func (s *State) Install(ctx context.Context, c *models.Cluster) (*UpdateReply, error) {
	state, err := s.getCurrentState(swag.StringValue(c.Status))
	if err != nil {
		return nil, err
	}
	return state.Install(ctx, c)
}

func (s *State) DeregisterCluster(ctx context.Context, c *models.Cluster) (*UpdateReply, error) {
	state, err := s.getCurrentState(swag.StringValue(c.Status))
	if err != nil {
		return nil, err
	}
	return state.DeregisterCluster(ctx, c)
}
