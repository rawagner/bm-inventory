package cluster

import (
	context "context"
	"io/ioutil"
	"testing"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/google/uuid"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sirupsen/logrus"

	"github.com/filanov/bm-inventory/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("statemachine", func() {
	var (
		ctx        = context.Background()
		db         *gorm.DB
		state      API
		cluster    models.Cluster
		stateReply *UpdateReply
		stateErr   error
	)

	BeforeEach(func() {
		db = prepareDB()
		state = NewState(getTestLog(), db)
		id := strfmt.UUID(uuid.New().String())
		cluster = models.Cluster{
			Base: models.Base{
				ID: &id,
			},
			Status: swag.String("not a known state"),
		}
		Expect(db.Create(&cluster).Error).ShouldNot(HaveOccurred())
	})

	Context("unknown_cluster_state", func() {
		It("register_cluster", func() {
			stateReply, stateErr = state.RegisterCluster(ctx, &cluster)
		})

		It("update_cluster", func() {
			stateReply, stateErr = state.RefreshStatus(ctx, &cluster, db)
		})

		It("install_cluster", func() {
			stateReply, stateErr = state.Install(ctx, &cluster)
		})

		It("deregister_cluster", func() {
			stateReply, stateErr = state.DeregisterCluster(ctx, &cluster)
		})

		AfterEach(func() {
			Expect(stateReply).To(BeNil())
			Expect(stateErr).Should(HaveOccurred())
		})
	})

})

func prepareDB() *gorm.DB {
	db, err := gorm.Open("sqlite3", ":memory:")
	Expect(err).ShouldNot(HaveOccurred())
	db.AutoMigrate(&models.Cluster{})
	return db
}

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cluster state machine tests")
}

func getTestLog() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	return l
}
