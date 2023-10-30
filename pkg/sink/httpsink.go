package sink

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"github.com/numaproj-labs/numaserve/pkg/common"
	"io"
	"net/http"
	"time"

	sinksdk "github.com/numaproj/numaflow-go/pkg/sinker"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

type HttpSink struct {
	Logger       *zap.SugaredLogger
	HttpClient   *http.Client
	URL          string
	Method       string
	Retries      int
	Timeout      int
	Windowing    int
	SkipInsecure bool
	DropIfError  bool
	Headers      arrayFlags
}
type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func (hs *HttpSink) CreateHTTPClient() {
	//creating http client
	client := &http.Client{Timeout: time.Duration(hs.Timeout) * time.Second}
	if hs.SkipInsecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}
	hs.HttpClient = client
}

func (hs *HttpSink) sendHTTPRequest(url string, data io.Reader) error {
	req, err := http.NewRequest(hs.Method, url, data)
	if err != nil {
		return err
	}
	if hs.HttpClient == nil {
		return errors.New("HTTP Client is not initialized")
	}
	res, err := hs.HttpClient.Do(req)
	if err != nil {
		return err
	}
	if res != nil {
		if res.Body != nil {
			res.Body.Close()
		}
		hs.Logger.Infof("Response code: %d,", res.StatusCode)
	}
	return nil
}

func (hs *HttpSink) Sink(ctx context.Context, datumStreamCh <-chan sinksdk.Datum) sinksdk.Responses {
	ok := sinksdk.ResponsesBuilder()
	failed := sinksdk.ResponsesBuilder()

	for datum := range datumStreamCh {
		//TODO Need to implemente parallel sending request
		//data := bytes.NewReader(datum.Value())
		mlReq := common.MLServeRequest{}
		hs.Logger.Infof("MLReq: %s", mlReq.ToJSON())
		err := json.Unmarshal(datum.Value(), &mlReq)
		if err != nil {
			hs.Logger.Errorf("HTTP Request failed. %v", err)
			failed = failed.Append(sinksdk.ResponseFailure(datum.ID(), "failed to forward message"))
			continue
		}
		hs.Logger.Infof("MLReq: %s", string(mlReq.Payload))
		rsp := common.MLServeResponse{
			Status:  200,
			Message: "",
			Payload: mlReq.Payload,
		}
		rspbyte, err := json.Marshal(rsp)
		if err != nil {
			hs.Logger.Errorf("HTTP Request failed. %v", err)
			failed = failed.Append(sinksdk.ResponseFailure(datum.ID(), "failed to forward message"))
			continue
		}
		backoff := wait.Backoff{
			Steps:    hs.Retries,
			Duration: 10 * time.Second,
			Factor:   2,
		}
		retryError := wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (done bool, err error) {
			err = hs.sendHTTPRequest(mlReq.GetDestination(), bytes.NewReader(rspbyte))
			if err != nil {
				hs.Logger.Errorf("HTTP Request failed. %v", err)
				return false, nil
			}
			return true, nil
		})
		if retryError != nil {
			hs.Logger.Errorf("HTTP Request failed. Error : %v", retryError)
			if hs.DropIfError {
				hs.Logger.Warn("Dropping messages due to failure")
				return ok
			}
			failed = failed.Append(sinksdk.ResponseFailure(datum.ID(), "failed to forward message"))
		}
		ok = ok.Append(sinksdk.ResponseOK(datum.ID()))
	}
	if len(failed) > 0 {
		return failed
	}
	hs.Logger.Info("HTTP Request send successfully")
	return ok
}

//func main() {
//	var metricPort int
//	labels := flag2.MapFlag{}
//	logger := logging.NewLogger().Named("http-sink")
//	hs := HttpSink{Logger: logger}
//	flag.StringVar(&hs.URL, "url", "", "URL")
//	flag.StringVar(&hs.Method, "method", "GET", "HTTP Method")
//	flag.IntVar(&hs.Retries, "retries", 3, "Request Retries")
//	flag.IntVar(&hs.Timeout, "timeout", 30, "Request Timeout in seconds")
//	flag.BoolVar(&hs.SkipInsecure, "insecure", false, "Skip TLS verify")
//	flag.BoolVar(&hs.DropIfError, "dropIfError", false, "Messages will drop after retry")
//	flag.Var(&hs.Headers, "headers", "HTTP Headers")
//	flag.IntVar(&metricPort, "udsinkMetricsPort", 9090, "UDSink Metrics Port")
//	flag.Var(&labels, "udsinkMetricsLabels", "UDSink Metrics Labels E.g: label=val1,label1=val2")
//	// Parse the flag
//	flag.Parse()
//
//	hs.Logger.Infof("Metrics publisher initialized with port=%d", metricPort)
//	//creating http client
//	hs.CreateHTTPClient()
//	hs.Logger.Info("HTTP Sink starting successfully with args %v", hs)
//	err := sinksdk.NewServer(&hs).Start(context.Background())
//	if err != nil {
//		log.Panic("Failed to start sink function server: ", err)
//	}
//}
