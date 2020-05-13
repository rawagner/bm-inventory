package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/filanov/bm-inventory/internal/bminventory"
	"github.com/filanov/bm-inventory/internal/cluster"
	"github.com/filanov/bm-inventory/internal/hardware"
	"github.com/filanov/bm-inventory/internal/host"
	"github.com/filanov/bm-inventory/models"
	"github.com/filanov/bm-inventory/pkg/job"
	"github.com/filanov/bm-inventory/pkg/requestid"
	"github.com/filanov/bm-inventory/pkg/thread"
	"github.com/filanov/bm-inventory/restapi"
)

func init() {
	strfmt.MarshalFormat = strfmt.ISO8601LocalTime
}

var Options struct {
	BMConfig               bminventory.Config
	DBHost                 string `envconfig:"DB_HOST" default:"mariadb"`
	DBPort                 string `envconfig:"DB_PORT" default:"3306"`
	HWValidatorConfig      hardware.ValidatorCfg
	JobConfig              job.Config
	InstructionConfig      host.InstructionConfig
	ClusterMonitorTimeLoop string `envconfig:"CLUSTER_MONITOR_TIME_LOOP" default:"500"` // Microsecond
}

func main() {
	log := logrus.New()
	err := envconfig.Process("myapp", &Options)
	if err != nil {
		log.Fatal(err.Error())
	}

	port := flag.String("port", "8090", "define port that the service will listen to") //nolint:staticcheck
	flag.Parse()

	log.Println("Starting bm service")

	db, err := gorm.Open("mysql",
		fmt.Sprintf("admin:admin@tcp(%s:%s)/installer?charset=utf8&parseTime=True&loc=Local",
			Options.DBHost, Options.DBPort))

	if err != nil {
		log.Fatal("Fail to connect to DB, ", err)
	}
	defer db.Close()

	scheme := runtime.NewScheme()
	if err = clientgoscheme.AddToScheme(scheme); err != nil {
		log.Fatal("Failed to add K8S scheme", err)
	}

	kclient, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme}) //nolint:staticcheck
	if err != nil {
		log.Fatal("failed to create client:", err)
	}

	if err = db.AutoMigrate(&models.Host{}, &models.Cluster{}).Error; err != nil {
		log.Fatal("failed to auto migrate, ", err)
	}

	hwValidator := hardware.NewValidator(Options.HWValidatorConfig)
	instructionApi := host.NewInstructionManager(log, db, hwValidator, Options.InstructionConfig)
	hostApi := host.NewManager(log.WithField("pkg", "host-state"), db, hwValidator, instructionApi)
	clusterApi := cluster.NewManager(log.WithField("pkg", "cluster-monitor"), db)

	clusterMonitor := thread.New(log, "State Monitor", time.Duration(500)*time.Microsecond, clusterApi.ClusterMonitoring)
	clusterMonitor.Start()
	defer clusterMonitor.Stop()

	jobApi := job.New(log.WithField("pkg", "k8s-job-wrapper"), kclient, Options.JobConfig)
	bm := bminventory.NewBareMetalInventory(db, log.WithField("pkg", "Inventory"), hostApi, clusterApi, Options.BMConfig, jobApi)
	h, err := restapi.Handler(restapi.Config{
		InstallerAPI: bm,
		Logger:       log.Printf,
	})
	h = requestid.Middleware(h)
	if err != nil {
		log.Fatal("Failed to init rest handler,", err)
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", swag.StringValue(port)), h))
}
