package s3

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cyverse/s3rods/commons"
	"github.com/cyverse/s3rods/irods"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// S3Service is a service object
type S3Service struct {
	config          *commons.Config
	irodsController *irods.IrodsController
	address         string
	router          *gin.Engine
	httpServer      *http.Server
}

// Start starts a new S3 service
func Start(config *commons.Config, irodsController *irods.IrodsController) (*S3Service, error) {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
		"function": "Start",
	})

	addr := fmt.Sprintf(":%d", config.Port)
	router := gin.Default()

	service := &S3Service{
		config:          config,
		irodsController: irodsController,
		address:         addr,
		router:          router,
		httpServer: &http.Server{
			Addr:    addr,
			Handler: router,
		},
	}

	// setup HTTP request router
	service.setupRouter()

	fmt.Printf("Starting S3 service at %s\n", service.address)
	logger.Infof("Starting S3 service at %s", service.address)
	// listen and serve in background
	go func() {
		err := service.httpServer.ListenAndServe()
		if err != nil {
			logger.Fatal(err)
		}
	}()

	return service, nil
}

// Stop stops the service
func (service *S3Service) Stop() error {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
		"struct":   "S3Service",
		"function": "Stop",
	})

	logger.Infof("Stopping S3 service\n")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := service.httpServer.Shutdown(ctx)
	if err != nil {
		logger.Error(err)
		return err
	}
	logger.Infof("Stopped S3 service\n")

	return nil
}
