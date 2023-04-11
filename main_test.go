package common

import (
	"testing"

	"github.com/zy21812/common/resp"
)

func EchoTest(t *testing.T) {
	e := resp.GetEcho()
	e.GET("/api/", func(c *resp.Context) error {
		return c.String(200, "ok")
	})
}
