package bminventory

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/pkg/errors"

	"github.com/filanov/bm-inventory/internal/cluster"
	"github.com/filanov/bm-inventory/internal/host"
	"github.com/filanov/bm-inventory/models"
	"github.com/filanov/bm-inventory/pkg/job"
	"github.com/filanov/bm-inventory/restapi/operations/inventory"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

func TestValidator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "inventory_test")
}

func prepareDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", ":memory:")
	Expect(err).ShouldNot(HaveOccurred())
	//db = db.Debug()
	db.AutoMigrate(&models.Cluster{}, &models.Host{})
	return db
}

func getTestLog() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	return l
}

func strToUUID(s string) *strfmt.UUID {
	u := strfmt.UUID(s)
	return &u
}

var _ = Describe("GenerateClusterISO", func() {
	var (
		bm      *bareMetalInventory
		cfg     Config
		db      *gorm.DB
		ctx     = context.Background()
		ctrl    *gomock.Controller
		mockJob *job.MockAPI
	)

	BeforeEach(func() {
		Expect(envconfig.Process("test", &cfg)).ShouldNot(HaveOccurred())
		ctrl = gomock.NewController(GinkgoT())
		db = prepareDB()
		mockJob = job.NewMockAPI(ctrl)
		bm = NewBareMetalInventory(db, getTestLog(), nil, nil, cfg, mockJob)
	})

	registerCluster := func() *models.Cluster {
		clusterId := strfmt.UUID(uuid.New().String())
		cluster := models.Cluster{
			Base: models.Base{
				ID: &clusterId,
			},
		}
		Expect(db.Create(&cluster).Error).ShouldNot(HaveOccurred())
		return &cluster
	}

	It("success", func() {
		clusterId := registerCluster().ID
		mockJob.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockJob.EXPECT().Monitor(gomock.Any(), gomock.Any(), defaultJobNamespace).Return(nil).Times(1)
		generateReply := bm.GenerateClusterISO(ctx, inventory.GenerateClusterISOParams{
			ClusterID:         *clusterId,
			ImageCreateParams: &models.ImageCreateParams{},
		})
		Expect(generateReply).Should(BeAssignableToTypeOf(inventory.NewGenerateClusterISOCreated()))
	})

	It("success with proxy", func() {
		clusterId := registerCluster().ID
		mockJob.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockJob.EXPECT().Monitor(gomock.Any(), gomock.Any(), defaultJobNamespace).Return(nil).Times(1)
		generateReply := bm.GenerateClusterISO(ctx, inventory.GenerateClusterISOParams{
			ClusterID:         *clusterId,
			ImageCreateParams: &models.ImageCreateParams{ProxyURL: "http://1.1.1.1:1234"},
		})
		Expect(generateReply).Should(BeAssignableToTypeOf(inventory.NewGenerateClusterISOCreated()))
	})
	It("cluster_not_exists", func() {
		generateReply := bm.GenerateClusterISO(ctx, inventory.GenerateClusterISOParams{
			ClusterID:         strfmt.UUID(uuid.New().String()),
			ImageCreateParams: &models.ImageCreateParams{},
		})
		Expect(generateReply).Should(BeAssignableToTypeOf(inventory.NewGenerateClusterISONotFound()))
	})

	It("failed_to_create_job", func() {
		clusterId := registerCluster().ID
		mockJob.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error")).Times(1)
		generateReply := bm.GenerateClusterISO(ctx, inventory.GenerateClusterISOParams{
			ClusterID:         *clusterId,
			ImageCreateParams: &models.ImageCreateParams{},
		})
		Expect(generateReply).Should(BeAssignableToTypeOf(inventory.NewGenerateClusterISOInternalServerError()))
	})

	It("job_failed", func() {
		clusterId := registerCluster().ID
		mockJob.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockJob.EXPECT().Monitor(gomock.Any(), gomock.Any(), defaultJobNamespace).Return(fmt.Errorf("error")).Times(1)
		generateReply := bm.GenerateClusterISO(ctx, inventory.GenerateClusterISOParams{
			ClusterID:         *clusterId,
			ImageCreateParams: &models.ImageCreateParams{},
		})
		Expect(generateReply).Should(BeAssignableToTypeOf(inventory.NewGenerateClusterISOInternalServerError()))
	})

	AfterEach(func() {
		ctrl.Finish()
		db.Close()
	})

})

var _ = Describe("GetNextSteps", func() {
	var (
		bm          *bareMetalInventory
		cfg         Config
		db          *gorm.DB
		ctx         = context.Background()
		ctrl        *gomock.Controller
		mockHostApi *host.MockAPI
	)

	BeforeEach(func() {
		Expect(envconfig.Process("test", &cfg)).ShouldNot(HaveOccurred())
		ctrl = gomock.NewController(GinkgoT())
		db = prepareDB()
		mockHostApi = host.NewMockAPI(ctrl)
		bm = NewBareMetalInventory(db, getTestLog(), mockHostApi, nil, cfg, nil)
	})

	It("get_next_steps_unknown_host", func() {
		clusterId := strToUUID(uuid.New().String())
		unregistered_hostID := strToUUID(uuid.New().String())

		generateReply := bm.GetNextSteps(ctx, inventory.GetNextStepsParams{
			ClusterID: *clusterId,
			HostID:    *unregistered_hostID,
		})
		Expect(generateReply).Should(BeAssignableToTypeOf(inventory.NewGetNextStepsNotFound()))
	})

	It("get_next_steps_success", func() {
		clusterId := strToUUID(uuid.New().String())
		hostId := strToUUID(uuid.New().String())
		host := models.Host{
			Base: models.Base{
				ID: hostId,
			},
			ClusterID: *clusterId,
			Status:    swag.String("discovering"),
		}
		Expect(db.Create(&host).Error).ShouldNot(HaveOccurred())

		var err error
		expectedStepsReply := models.Steps{&models.Step{StepType: models.StepTypeHardwareInfo},
			&models.Step{StepType: models.StepTypeConnectivityCheck}}
		mockHostApi.EXPECT().GetNextSteps(gomock.Any(), gomock.Any()).Return(expectedStepsReply, err)
		reply := bm.GetNextSteps(ctx, inventory.GetNextStepsParams{
			ClusterID: *clusterId,
			HostID:    *hostId,
		})
		Expect(reply).Should(BeAssignableToTypeOf(inventory.NewGetNextStepsOK()))
		stepsReply := reply.(*inventory.GetNextStepsOK).Payload
		expectedStepsType := []models.StepType{models.StepTypeHardwareInfo, models.StepTypeConnectivityCheck}
		Expect(stepsReply).To(HaveLen(len(expectedStepsType)))
		for i, step := range stepsReply {
			Expect(step.StepType).Should(Equal(expectedStepsType[i]))
		}
	})

	AfterEach(func() {
		ctrl.Finish()
		db.Close()
	})
})

var _ = Describe("UpdateHostInstallProgress", func() {
	var (
		bm          *bareMetalInventory
		cfg         Config
		db          *gorm.DB
		ctx         = context.Background()
		ctrl        *gomock.Controller
		mockHostApi *host.MockAPI
	)

	BeforeEach(func() {
		Expect(envconfig.Process("test", &cfg)).ShouldNot(HaveOccurred())
		ctrl = gomock.NewController(GinkgoT())
		db = prepareDB()
		mockHostApi = host.NewMockAPI(ctrl)
		bm = NewBareMetalInventory(db, getTestLog(), mockHostApi, nil, cfg, nil)
	})

	Context("host exists", func() {
		var hostID, clusterID strfmt.UUID
		BeforeEach(func() {
			hostID = strfmt.UUID(uuid.New().String())
			clusterID = strfmt.UUID(uuid.New().String())
			err := db.Create(&models.Host{
				Base:      models.Base{ID: &hostID},
				ClusterID: clusterID,
			}).Error
			Expect(err).ShouldNot(HaveOccurred())

		})

		It("success", func() {
			mockHostApi.EXPECT().UpdateInstallProgress(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			reply := bm.UpdateHostInstallProgress(ctx, inventory.UpdateHostInstallProgressParams{
				ClusterID:                 clusterID,
				HostInstallProgressParams: "some progress",
				HostID:                    hostID,
			})
			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewUpdateHostInstallProgressOK()))
		})

		It("update_failed", func() {
			mockHostApi.EXPECT().UpdateInstallProgress(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
			reply := bm.UpdateHostInstallProgress(ctx, inventory.UpdateHostInstallProgressParams{
				ClusterID:                 clusterID,
				HostInstallProgressParams: "some progress",
				HostID:                    hostID,
			})
			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewUpdateHostInstallProgressOK()))
		})
	})

	It("host_dont_exist", func() {
		reply := bm.UpdateHostInstallProgress(ctx, inventory.UpdateHostInstallProgressParams{
			ClusterID:                 strfmt.UUID(uuid.New().String()),
			HostInstallProgressParams: "some progress",
			HostID:                    strfmt.UUID(uuid.New().String()),
		})
		Expect(reply).Should(BeAssignableToTypeOf(inventory.NewUpdateHostInstallProgressOK()))
	})

	AfterEach(func() {
		ctrl.Finish()
		db.Close()
	})
})

var _ = Describe("cluster", func() {
	masterHostId1 := strfmt.UUID(uuid.New().String())
	masterHostId2 := strfmt.UUID(uuid.New().String())
	masterHostId3 := strfmt.UUID(uuid.New().String())

	var (
		bm             *bareMetalInventory
		cfg            Config
		db             *gorm.DB
		ctx            = context.Background()
		ctrl           *gomock.Controller
		mockHostApi    *host.MockAPI
		mockClusterApi *cluster.MockAPI
		mockJob        *job.MockAPI
		clusterID      strfmt.UUID
	)

	addHost := func(hostId strfmt.UUID, role string, state string, clusterId strfmt.UUID, db *gorm.DB) models.Host {
		host := models.Host{
			Base: models.Base{
				ID: &hostId,
			},
			ClusterID: clusterId,
			Status:    swag.String(state),
			Role:      role,
		}
		Expect(db.Create(&host).Error).ShouldNot(HaveOccurred())
		return host
	}
	getDisk := func() *models.BlockDevice {
		disk := models.BlockDevice{DeviceType: "loop", Fstype: "test", MajorDeviceNumber: 7, MinorDeviceNumber: 0, Mountpoint: "/sysroot", Name: "loop0", ReadOnly: true, RemovableDevice: 1, Size: 0}
		return &disk
	}
	setDefaultInstall := func(mockClusterApi *cluster.MockAPI) {
		mockClusterApi.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	}
	setDefaultGetMasterNodesIds := func(mockClusterApi *cluster.MockAPI) {
		mockClusterApi.EXPECT().GetMasterNodesIds(gomock.Any(), gomock.Any(), gomock.Any()).Return([]*strfmt.UUID{&masterHostId1, &masterHostId2, &masterHostId3}, nil)
	}
	setDefaultJobCreate := func(mockJobApi *job.MockAPI) {
		mockJob.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	}
	setDefaultJobMaonitor := func(mockJobApi *job.MockAPI) {
		mockJob.EXPECT().Monitor(gomock.Any(), gomock.Any(), defaultJobNamespace).Return(nil).Times(1)
	}
	setDefaultHostInstall := func(mockClusterApi *cluster.MockAPI) {
		mockHostApi.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	}
	setDefaultHostGetHostValidDisks := func(mockClusterApi *cluster.MockAPI) {
		mockHostApi.EXPECT().GetHostValidDisks(gomock.Any()).Return([]*models.BlockDevice{getDisk()}, nil).AnyTimes()
	}

	BeforeEach(func() {
		Expect(envconfig.Process("test", &cfg)).ShouldNot(HaveOccurred())
		ctrl = gomock.NewController(GinkgoT())
		db = prepareDB()
		mockJob = job.NewMockAPI(ctrl)
		mockClusterApi = cluster.NewMockAPI(ctrl)
		mockHostApi = host.NewMockAPI(ctrl)
		bm = NewBareMetalInventory(db, getTestLog(), mockHostApi, mockClusterApi, cfg, mockJob)

	})

	Context("Install", func() {
		BeforeEach(func() {
			clusterID = strfmt.UUID(uuid.New().String())
			err := db.Create(&models.Cluster{
				Base: models.Base{ID: &clusterID},
			}).Error
			Expect(err).ShouldNot(HaveOccurred())

			addHost(masterHostId1, "master", "known", clusterID, db)
			addHost(masterHostId2, "master", "known", clusterID, db)
			addHost(masterHostId3, "master", "known", clusterID, db)
		})

		It("success", func() {

			setDefaultInstall(mockClusterApi)
			setDefaultGetMasterNodesIds(mockClusterApi)

			setDefaultJobCreate(mockJob)
			setDefaultJobMaonitor(mockJob)

			setDefaultHostInstall(mockClusterApi)
			setDefaultHostGetHostValidDisks(mockClusterApi)

			reply := bm.InstallCluster(ctx, inventory.InstallClusterParams{
				ClusterID: clusterID,
			})

			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewInstallClusterOK()))
		})
		It("cluster failed to update", func() {
			mockClusterApi.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.Errorf("cluster has a error"))
			reply := bm.InstallCluster(ctx, inventory.InstallClusterParams{
				ClusterID: clusterID,
			})
			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewInstallClusterConflict()))

		})
		It("host failed to install", func() {

			setDefaultInstall(mockClusterApi)
			setDefaultGetMasterNodesIds(mockClusterApi)

			mockHostApi.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.Errorf("host has a error")).AnyTimes()
			setDefaultHostGetHostValidDisks(mockClusterApi)

			reply := bm.InstallCluster(ctx, inventory.InstallClusterParams{
				ClusterID: clusterID,
			})
			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewInstallClusterConflict()))

		})
		It("GetMasterNodesIds fails", func() {

			setDefaultInstall(mockClusterApi)
			mockClusterApi.EXPECT().GetMasterNodesIds(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*strfmt.UUID{&masterHostId1, &masterHostId2, &masterHostId3}, errors.Errorf("nop"))

			setDefaultHostInstall(mockClusterApi)
			setDefaultHostGetHostValidDisks(mockClusterApi)

			reply := bm.InstallCluster(ctx, inventory.InstallClusterParams{
				ClusterID: clusterID,
			})

			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewInstallClusterInternalServerError()))
		})
		It("GetMasterNodesIds returns empty list", func() {

			setDefaultInstall(mockClusterApi)
			mockClusterApi.EXPECT().GetMasterNodesIds(gomock.Any(), gomock.Any(), gomock.Any()).
				Return([]*strfmt.UUID{&masterHostId1, &masterHostId2, &masterHostId3}, errors.Errorf("nop"))

			setDefaultHostInstall(mockClusterApi)
			setDefaultHostGetHostValidDisks(mockClusterApi)

			reply := bm.InstallCluster(ctx, inventory.InstallClusterParams{
				ClusterID: clusterID,
			})

			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewInstallClusterInternalServerError()))
		})
		//GetHostValidDisks
		It("GetHostValidDisks returns err", func() {

			setDefaultInstall(mockClusterApi)
			setDefaultGetMasterNodesIds(mockClusterApi)

			setDefaultHostInstall(mockClusterApi)
			mockHostApi.EXPECT().GetHostValidDisks(gomock.Any()).Return(nil, errors.Errorf("you fail")).AnyTimes()

			reply := bm.InstallCluster(ctx, inventory.InstallClusterParams{
				ClusterID: clusterID,
			})

			Expect(reply).Should(BeAssignableToTypeOf(inventory.NewInstallClusterInternalServerError()))
		})
	})
	AfterEach(func() {
		ctrl.Finish()
		db.Close()
	})
})
