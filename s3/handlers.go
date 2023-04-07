package s3

import (
	"net/http"

	"github.com/cyverse/s3rods/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

// setupRouter setup http request router
func (service *S3Service) setupRouter() {
	service.router.GET("/ping", service.handlePing)
	service.router.GET("/", service.handleRoot)
}

func (service *S3Service) handlePing(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
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

func (service *S3Service) setResponseHeader(c *gin.Context) {
	header := c.Writer.Header()
	header.Set("Server", "S3Rods")
	header.Set("X-Amz-Request-Id", xid.New().String()) // new iD
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Xss-Protection", "1; mode=block")
}

func (service *S3Service) authenticateUser(c *gin.Context) (*AWSCredential, error) {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
		"struct":   "S3Service",
		"function": "authenticateUser",
	})

	credential := getCredential(c.Request)
	if credential == nil {
		return nil, xerrors.Errorf("failed to get credential from request")
	}

	secretKey, err := service.irodsController.GetUserSecretKey(credential.Username)
	if err != nil {
		return nil, err
	}

	// auth
	checked, err := checkSignature(c.Request, secretKey)
	if err != nil {
		return nil, err
	}

	logger.Infof("authenticateUser %s result: %t", credential.Username, checked)
	return credential, nil
}

func (service *S3Service) handleRoot(c *gin.Context) {
	logger := log.WithFields(log.Fields{
		"package":  "s3",
		"struct":   "S3Service",
		"function": "handleRoot",
	})

	logger.Infof("access request to %s", c.Request.URL)

	credential, err := service.authenticateUser(c)
	if err != nil {
		c.XML(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	rootDirStats, err := service.irodsController.ListRootDirStats(credential.Username)
	if err != nil {
		c.XML(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	service.setResponseHeader(c)
	awsUser := types.NewAwsUser(credential.Username)

	buckets := make([]types.Bucket, len(rootDirStats))
	for bucketID, rootDirStat := range rootDirStats {
		bucket := types.NewBucket(rootDirStat.Name, rootDirStat.CreateTime)
		buckets[bucketID] = bucket
	}

	logger.Debugf("buckets %v", buckets)

	output := types.ListBucketsOutput{
		Owner:   awsUser,
		Buckets: buckets,
	}
	c.XML(http.StatusOK, output)
}
