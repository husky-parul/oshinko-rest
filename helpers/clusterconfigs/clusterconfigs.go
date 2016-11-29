package clusterconfigs

import (
	"errors"
	"strconv"
	"github.com/radanalyticsio/oshinko-rest/models"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/api"
	"fmt"
)


var defaultConfig models.NewClusterConfig = models.NewClusterConfig{
								MasterCount: 1,
	                                                        WorkerCount: 1,
								Name: "default",
								SparkMasterConfig: "",
								SparkWorkerConfig: ""}

const Defaultname = "default"
const failOnMissing = true
const allowMissing = false
const DefaultConfigPath = "/etc/oshinko-cluster-configs/"

const MasterCountMustBeOne = "Cluster configuration must have a masterCount of 1"
const WorkerCountMustBeAtLeastOne = "Cluster configuration may not have a workerCount less than 1"
const NamedConfigDoesNotExist = "Named config '%s' does not exist"
const ErrorWhileProcessing = "Error while processing %s: %s"

// This function is meant to support testability
func GetDefaultConfig() models.NewClusterConfig {
	return defaultConfig
}

func assignConfig(res *models.NewClusterConfig, src models.NewClusterConfig) {
	if src.MasterCount != 0 {
		res.MasterCount = src.MasterCount
	}
	if src.WorkerCount != 0 {
		res.WorkerCount = src.WorkerCount
	}

	if src.SparkMasterConfig != "" {
		res.SparkMasterConfig = src.SparkMasterConfig
	}
	if src.SparkWorkerConfig != "" {
		res.SparkWorkerConfig = src.SparkWorkerConfig
	}
}

func checkConfiguration(config models.NewClusterConfig) error {
	var err error
	if config.MasterCount != 1 {
		err = errors.New(MasterCountMustBeOne)
	} else if config.WorkerCount < 1 {
		err = errors.New(WorkerCountMustBeAtLeastOne)
	}
	return err
}


func getInt64(value, configmapname string) (int64, error) {
	i, err := strconv.Atoi(value)
	if err != nil {
		err = errors.New(fmt.Sprintf(ErrorWhileProcessing, configmapname, errors.New("expected integer")))
	}
	return int64(i), err
}

func process(config *models.NewClusterConfig, name, value, configmapname string) error {

	var err error

	// At present we only have a single level of configs, but if/when we have
	// nested configs then we would descend through the levels beginning here with
	// the first element in the name
	switch name {
	case "mastercount":
		config.MasterCount, err = getInt64(value, configmapname + ".mastercount")
	case "workercount":
		config.WorkerCount, err = getInt64(value, configmapname + ".workercount")
	case "sparkmasterconfig":
                config.SparkMasterConfig = value
	case "sparkworkerconfig":
                config.SparkWorkerConfig = value
	}
	return err
}

func checkForConfigMap(name string, failOnMissing bool, cm kclient.ConfigMapsInterface) (*api.ConfigMap, error) {
	cmap, err := cm.Get(name)
	if (cmap == nil || len(cmap.Data) == 0) && failOnMissing == false {
		return cmap, nil
	}
	return cmap, err
}

func readConfig(name string, res *models.NewClusterConfig, failOnMissing bool, cm kclient.ConfigMapsInterface) (err error) {
        cmap, err := checkForConfigMap(name, failOnMissing, cm)
	if err == nil && cmap != nil {
                for n, v := range (cmap.Data) {
			err = process(res, n, v, name)
			if err != nil {
				break
			}
		}
	}
	return err
}

func loadConfig(name string, cm kclient.ConfigMapsInterface) (res models.NewClusterConfig, err error) {
	// If the default config has been modified use those mods.
	res = defaultConfig
	err = readConfig(Defaultname, &res, allowMissing, cm)
	if err == nil && name != "" && name != Defaultname {
		err = readConfig(name, &res, failOnMissing, cm)
	}
	return res, err
}

func GetClusterConfig(config *models.NewClusterConfig, cm kclient.ConfigMapsInterface) (res models.NewClusterConfig, err error) {
        var name string = ""
	if config != nil {
	   name = config.Name
	}
	res, err = loadConfig(name, cm)
	if err == nil && config != nil {
		assignConfig(&res, *config)
	}

	// Check that the final configuration is valid
	if err == nil {
		err = checkConfiguration(res)
	}
	return res, err
}
