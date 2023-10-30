package frontline

import (
	"context"
	"fmt"
	"github.com/caitlinelfring/go-env-default"
	"github.com/numaproj-labs/numaserve/pkg/common"
	"github.com/slok/go-http-metrics/middleware"
	"github.com/slok/go-http-metrics/middleware/std"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type Service struct {
	port          int
	tlsSkipVerify bool
	requestQueue  map[string]*RequestSession
	sync          bool
	backendURL    string
	callbackURL   string
	rwlock        sync.RWMutex
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		// Set example variable
		c.Set("example", "12345")

		// before request
		c.Next()
		// after request
		latency := time.Since(t)
		log.Print(latency)

		// access the status we are sending
		status := c.Writer.Status()
		log.Println(status)
	}
}

func NewFrontLineService(port int, backendURL string) *Service {
	return &Service{port: port, backendURL: backendURL, requestQueue: make(map[string]*RequestSession), callbackURL: getCallBackURL()}
}

func getCallBackURL() string {
	podName := env.GetDefault("POD_NAME", "")
	serviceName := env.GetDefault("SERVICE_NAME", "")
	namespaceName := env.GetDefault("NAMESPACE", "")
	callback := fmt.Sprintf("http://%s.%s.%s.svc.cluster.local:8443/callback", podName, serviceName, namespaceName)
	fmt.Println("callback", callback)
	return callback
}

func (fs *Service) Run(ctx context.Context) {
	r := gin.New()
	r.Use(Logger())
	mdlw := middleware.New(middleware.Config{
		Recorder: common.NewRecorder(common.Config{
			StatusCodeLabel: "status",
			HandlerIDLabel:  "uri",
		}),
	})

	r.POST("/predict", fs.synchandle)
	r.POST("/callback/:reqId", fs.handleResponse)
	handler := std.Handler("", mdlw, r)
	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", fs.port), handler)
	if err != nil {
		log.Fatal(err)
	}
}
