package utils

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

type HandlerFunc func(c *gin.Context) (interface{}, error)

func Wrapper(handler HandlerFunc) func(c *gin.Context) {
	return func(cc *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("unknown error: %s", err.Error())
				}
				fmt.Println("handle a panic", string(debug.Stack()))
				cc.JSON(500, err.Error())
			}
		}()
		res, err := handler(cc)
		if err != nil {
			fmt.Println("failed to handler request")
			cc.JSON(500, err.Error())
			return
		}
		fmt.Println("process success", res)
		cc.JSON(http.StatusOK, res)
	}
}
