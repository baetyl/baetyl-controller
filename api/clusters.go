package api

import (
	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/baetyl/baetyl-controller/kube/apis/baetyl/v1alpha1"
)

func (api *AdminAPI) CreateCluster(c *gin.Context) (interface{}, error) {
	info := v1alpha1.Cluster{}
	err := c.ShouldBindJSON(&info)
	if err != nil {
		return nil, err
	}
	info.Namespace = c.GetString("namespace")
	return api.cli.CreateCluster(&info)
}

func (api *AdminAPI) UpdateCluster(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	if _, err := api.cli.GetCluster(ns, n); err != nil {
		return nil, ErrNotFound
	}
	info := v1alpha1.Cluster{}
	err := c.ShouldBindJSON(&info)
	if err != nil {
		return nil, err
	}
	info.Namespace = ns
	return api.cli.UpdateCluster(&info)
}

func (api *AdminAPI) DeleteCluster(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	return nil, api.cli.DeleteCluster(ns, n)
}

func (api *AdminAPI) GetCluster(c *gin.Context) (interface{}, error) {
	ns, n := c.GetString("namespace"), c.Param("name")
	return api.cli.GetCluster(ns, n)
}

func (api *AdminAPI) ListClusters(c *gin.Context) (interface{}, error) {
	return api.cli.ListClusters(c.GetString("namespace"), labels.NewSelector())
}
