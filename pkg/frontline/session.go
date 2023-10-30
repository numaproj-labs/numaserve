package frontline

import (
	"github.com/gin-gonic/gin"
	"github.com/numaproj-labs/numaserve/pkg/common"
)

type RequestSession struct {
	id   string
	resp *chan common.MLServeResponse
}

func NewRequestSession(resp *chan common.MLServeResponse) *RequestSession {
	return &RequestSession{resp: resp}
}

func (rs *RequestSession) Received(response *common.MLServeResponse) {
	*rs.resp <- *response
}
func (rs *RequestSession) wait(ctx *gin.Context) {
	for data := range *rs.resp {
		ctx.JSON(data.Status, string(data.Payload))
		break
	}
}
