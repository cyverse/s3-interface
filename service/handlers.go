package service

import (
	"io"
	"net/http"
	"time"

	"github.com/cyverse/s3-interface/types"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
)

// setupRouter setup http request router
func (service *S3Service) setupRouter() {
	service.router.GET("/ping", service.handlePing)
	service.router.GET("/", service.handleRoot)
}

func (service *S3Service) handlePing(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"package":  "rest",
		"struct":   "S3Service",
		"function": "handlePing",
	})

	logger.Infof("access request to %s", c.Request.URL)

	type pingOutput struct {
		Message string `json:"message"`
	}

	output := pingOutput{
		Message: "pong",
	}
	c.JSON(http.StatusOK, output)
}

func (service *S3Service) setResponseHeader(header http.Header) {
	header.Set("Server", "S3-Interface")
	header.Set("X-Amz-Request-Id", xid.New().String()) // new iD
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Xss-Protection", "1; mode=block")
}

func (service *S3Service) handleRoot(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"package":  "rest",
		"struct":   "S3Service",
		"function": "handleRoot",
	})

	logger.Infof("access request to %s", c.Request.URL)

	// auth
	checked, err := checkSignature(c.Request, "xxx")
	if err != nil {
		c.XML(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	logger.Infof("authentication result: %t", checked)

	writeHeader := c.Writer.Header()
	service.setResponseHeader(writeHeader)

	output := types.ListBucketsOutput{
		Owner: types.AwsUser{
			ID:          "iychoi",
			DisplayName: "iychoi",
		},
		Buckets: []types.Bucket{
			{
				Name:         "bucket1",
				CreationDate: time.Now(),
			},
			{
				Name:         "bucket2",
				CreationDate: time.Now(),
			},
		},
	}
	c.XML(http.StatusOK, output)
}

func (service *S3Service) handleTest(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"package":  "rest",
		"struct":   "S3Service",
		"function": "handleTest",
	})

	logger.Infof("access request to %s", c.Request.URL)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	logger.Infof("body: %s", string(body))

	type pingOutput struct {
		Message string `json:"message"`
	}

	output := pingOutput{
		Message: "pong",
	}
	c.JSON(http.StatusOK, output)
}
