package cluster

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/filanov/bm-inventory/models"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
)

func NewInsufficientState(log logrus.FieldLogger, db *gorm.DB) *insufficientState {
	return &insufficientState{
		log: log,
		db:  db,
	}
}

type insufficientState baseState

var _ API = (*State)(nil)

func (i *insufficientState) RegisterCluster(ctx context.Context, c *models.Cluster) (*UpdateReply, error) {
	tx := i.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			i.log.Error("update cluster failed")
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		i.log.WithError(tx.Error).Error("failed to start transaction")
	}

	if err := tx.Preload("Hosts").Create(c).Error; err != nil {
		i.log.Errorf("Error registering cluster %s", c.Name)
		tx.Rollback()
		return &UpdateReply{
			State:     clusterStatusInsufficient,
			IsChanged: false,
		}, err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		i.log.WithError(err).Errorf("failed to commit cluster %s changes on installation", c.ID.String())
		return &UpdateReply{
			State:     clusterStatusInsufficient,
			IsChanged: false,
		}, err
	}

	return &UpdateReply{
		State:     clusterStatusInsufficient,
		IsChanged: true,
	}, nil
}

func (i *insufficientState) RefreshStatus(ctx context.Context, c *models.Cluster, db *gorm.DB) (*UpdateReply, error) {

	clusterIsReady, err := isClusterReady(c, db, i.log)
	if err != nil {
		return nil, errors.Errorf("unable to determine cluster %s hosts state ", c.ID)
	}

	if clusterIsReady {
		return updateState(clusterStatusReady, c, db)
	} else {
		i.log.Infof("Cluster %s does not have sufficient resources to be installed.", c.ID)
		return &UpdateReply{
			State:     clusterStatusInsufficient,
			IsChanged: false,
		}, nil
	}
}

func (i *insufficientState) Install(ctx context.Context, c *models.Cluster) (*UpdateReply, error) {
	return nil, errors.Errorf("unable to install cluster <%s> in <%s> status",
		c.ID, swag.StringValue(c.Status))
}

func (i *insufficientState) DeregisterCluster(ctx context.Context, c *models.Cluster) (*UpdateReply, error) {
	return deregisterCluster(c, i.db)
}
