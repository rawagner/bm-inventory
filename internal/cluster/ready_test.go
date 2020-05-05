package cluster

import (
	context "context"

	"github.com/filanov/bm-inventory/models"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("insufficient_state", func() {
	var (
		ctx         = context.Background()
		state       API
		db          *gorm.DB
		id          strfmt.UUID
		updateReply *UpdateReply
		updateErr   error
		cluster     models.Cluster
		host        models.Host
	)

	BeforeEach(func() {
		db = prepareDB()
		state = &Manager{ready: NewReadyState(getTestLog(), db)}

		id = strfmt.UUID(uuid.New().String())
		cluster = models.Cluster{
			Base: models.Base{
				ID: &id,
			},
			Status: swag.String(clusterStatusReady),
		}
		Expect(db.Create(&cluster).Error).ShouldNot(HaveOccurred())
		addInstallationRequirements(id, db)

		c := geCluster(*cluster.ID, db)
		Expect(swag.StringValue(c.Status)).Should(Equal(clusterStatusReady))

		addInstallationRequirements(id, db)
		println(swag.StringValue(cluster.Status))
		updateReply, updateErr = state.RefreshStatus(ctx, &cluster, db)

		Expect(updateErr).Should(BeNil())
		Expect(updateReply.State).Should(Equal(clusterStatusReady))
		cluster = *geCluster(*cluster.ID, db)
		Expect(swag.StringValue(cluster.Status)).Should(Equal(clusterStatusReady))
	})

	Context("refresh_state", func() {
		It("cluster is not satisfying the install requirements", func() {
			cluster.Status = swag.String(clusterStatusReady)
			Expect(db.Where("cluster_id = ?", cluster.ID).Delete(&models.Host{}).Error).NotTo(HaveOccurred())

			updateReply, updateErr = state.RefreshStatus(ctx, &cluster, db)

			Expect(updateErr).Should(BeNil())
			Expect(updateReply.State).Should(Equal(clusterStatusInsufficient))
			Expect(updateReply.IsChanged).Should(Equal(true))

			cluster = *geCluster(*cluster.ID, db)
			Expect(swag.StringValue(cluster.Status)).Should(Equal(clusterStatusInsufficient))

		})
	})

	Context("deregister", func() {
		It("unregister a ready cluster", func() {
			println(len(cluster.Hosts))
			updateReply, updateErr = state.DeregisterCluster(ctx, &cluster)
			Expect(updateErr).Should(BeNil())
			Expect(updateReply.State).Should(Equal("unregistered"))
			Expect(db.First(&cluster, "id = ?", cluster.ID).Error).Should(HaveOccurred())
			Expect(db.First(&host, "cluster_id = ?", cluster.ID).Error).Should(HaveOccurred())
		})
	})

	AfterEach(func() {

		db.Close()
		updateReply = nil
		updateErr = nil
	})
})
