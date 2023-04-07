package irods

import (
	"github.com/cyverse/s3rods/commons"
	log "github.com/sirupsen/logrus"
)

// IrodsController is a controller object
type IrodsController struct {
	config *commons.Config
}

// Start starts a new S3 service
func Start(config *commons.Config) (*IrodsController, error) {
	logger := log.WithFields(log.Fields{
		"package":  "irods",
		"function": "Start",
	})

	logger.Info("Starting IRODS controller")
	controller := &IrodsController{
		config: config,
	}

	return controller, nil
}

// Stop stops the service
func (controller *IrodsController) Stop() error {
	logger := log.WithFields(log.Fields{
		"package":  "irods",
		"struct":   "IRodsController",
		"function": "Stop",
	})

	logger.Infof("Stopping IRODS controller\n")

	logger.Infof("Stopped IRODS controller\n")

	return nil
}
