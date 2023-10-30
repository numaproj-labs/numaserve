package frontline

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/numaproj-labs/numaserve/pkg/common"

	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func (fs *Service) synchandle(ctx *gin.Context) {
	id := uuid.New().String()
	fmt.Println("UUID", id)
	respChan := make(chan common.MLServeResponse)
	req := NewRequestSession(&respChan)

	fmt.Println(fs.requestQueue)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Println(err)
	}
	backendRequest := common.MLServeRequest{
		Metadata: common.Metadata{Sender: fs.callbackURL, RequestID: id},
		Payload:  body,
	}
	fmt.Println(backendRequest)
	buffer := bytes.NewBuffer([]byte(backendRequest.ToJSON()))
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := http.Client{Transport: tr}
	rsp, err := client.Post(fs.backendURL, "application/json", buffer)
	if err != nil {
		fmt.Println(err)
		ctx.JSON(500, err)
	}
	//if rsp.StatusCode > 299 {
	fmt.Println("MLServeRequest sent successfully", rsp)
	//}
	fs.rwlock.Lock()
	fs.requestQueue[id] = req
	fs.rwlock.Unlock()
	//todo retry and error handle
	req.wait(ctx)

}

func (fs *Service) handleResponse(ctx *gin.Context) {
	requestId := ctx.Param("reqId")
	resp := &common.MLServeResponse{}
	fs.rwlock.Lock()
	requestSession, ok := fs.requestQueue[strings.TrimSpace(requestId)]
	delete(fs.requestQueue, strings.TrimSpace(requestId))
	fs.rwlock.Unlock()
	if !ok {
		fmt.Println("request is not found")
		return
	}
	//var reqBody []byte
	ByteBody, _ := ioutil.ReadAll(ctx.Request.Body)
	strBody := string(ByteBody)
	fmt.Println("Response:", strBody)
	err := json.Unmarshal([]byte(strBody), resp)
	fmt.Println(err)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("response", resp)
	requestSession.Received(resp)
	ctx.JSON(200, "")
}
