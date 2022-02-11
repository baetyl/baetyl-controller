package api

import (
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/baetyl/baetyl-controller/kube/apis/baetyl/v1alpha1"
)

func (api *AdminAPI) CreateTemplate(c *gin.Context) (interface{}, error) {
	info := v1alpha1.Template{}
	err := c.ShouldBindJSON(&info)
	if err != nil {
		return nil, err
	}
	info.Namespace = c.GetString("namespace")
	return api.cli.CreateTemplate(&info)
}

func (api *AdminAPI) UpdateTemplate(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	if _, err := api.cli.GetTemplate(ns, n); err != nil {
		return nil, ErrNotFound
	}
	info := v1alpha1.Template{}
	err := c.ShouldBindJSON(&info)
	if err != nil {
		return nil, err
	}
	info.Namespace = ns
	return api.cli.UpdateTemplate(&info)
}

func (api *AdminAPI) DeleteTemplate(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	return nil, api.cli.DeleteTemplate(ns, n)
}

func (api *AdminAPI) GetTemplate(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	return api.cli.GetTemplate(ns, n)
}

func (api *AdminAPI) ListTemplates(c *gin.Context) (interface{}, error) {
	return api.cli.ListTemplates(c.GetString("namespace"), labels.NewSelector())
}
