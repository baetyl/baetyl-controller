package api

import (
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/baetyl/baetyl-controller/kube/apis/baetyl/v1alpha1"
)

func (api *AdminAPI) CreateApply(c *gin.Context) (interface{}, error) {
	info := v1alpha1.Apply{}
	err := c.ShouldBindJSON(&info)
	if err != nil {
		return nil, err
	}
	info.Namespace = c.GetString("namespace")
	return api.cli.CreateApply(&info)
}

func (api *AdminAPI) UpdateApply(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	if _, err := api.cli.GetApply(ns, n); err != nil {
		return nil, ErrNotFound
	}
	info := v1alpha1.Apply{}
	err := c.ShouldBindJSON(&info)
	if err != nil {
		return nil, err
	}
	info.Namespace = ns
	return api.cli.UpdateApply(&info)
}

func (api *AdminAPI) DeleteApply(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	return nil, api.cli.DeleteApply(ns, n)
}

func (api *AdminAPI) GetApply(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	return api.cli.GetApply(ns, n)
}

func (api *AdminAPI) ListApplies(c *gin.Context) (interface{}, error) {
	return api.cli.ListApplies(c.GetString("namespace"), labels.NewSelector())
}
