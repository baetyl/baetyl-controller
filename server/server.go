package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/baetyl/baetyl-controller/api"
	"github.com/baetyl/baetyl-controller/utils"
)

type AdminServer struct {
	cfg    ServerConfig
	router *gin.Engine
	api    *api.AdminAPI
	server *http.Server
}

func NewServer(path string) (*AdminServer, error) {
	var cfg ServerConfig
	err := utils.LoadYAML(path, &cfg)
	if err != nil {
		return nil, err
	}

	router := gin.New()
	svr := &http.Server{
		Addr:           cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}
	if cfg.Server.Certificate.Cert != "" &&
		cfg.Server.Certificate.Key != "" &&
		cfg.Server.Certificate.CA != "" {
		t, err := utils.NewTLSConfigServer(utils.Certificate{
			CA:             cfg.Server.Certificate.CA,
			Cert:           cfg.Server.Certificate.Cert,
			Key:            cfg.Server.Certificate.Key,
			ClientAuthType: cfg.Server.Certificate.ClientAuthType,
		})
		if err != nil {
			return nil, err
		}
		svr.TLSConfig = t
	}
	return &AdminServer{
		cfg:    cfg,
		router: router,
		server: svr,
	}, nil
}

func (a *AdminServer) Run() {
	a.initRoute()
	if a.server.TLSConfig == nil {
		if err := a.server.ListenAndServe(); err != nil {
			fmt.Println("server http stopped", err.Error())
		}
	} else {
		if err := a.server.ListenAndServeTLS("", ""); err != nil {
			fmt.Println("server https stopped", err.Error())
		}
	}
}

// Close close server
func (a *AdminServer) Close() error {
	ctx, _ := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTime)
	return a.server.Shutdown(ctx)
}

func (a *AdminServer) SetAPI(api *api.AdminAPI) {
	a.api = api
}

func (a *AdminServer) initRoute() {
	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(200, "success")
	})

	// TODO
	a.router.Use(func(c *gin.Context) {
		c.Set("namespace", "default")
	})

	v1 := a.router.Group("v3")
	{
		object := v1.Group("/applies")
		object.POST("", utils.Wrapper(a.api.CreateApply))
		object.GET("", utils.Wrapper(a.api.ListApplies))
		object.GET("/:name", utils.Wrapper(a.api.GetApply))
		object.PUT("/:name", utils.Wrapper(a.api.UpdateApply))
		object.DELETE("/:name", utils.Wrapper(a.api.DeleteApply))
	}
	{
		object := v1.Group("/templates")
		object.POST("", utils.Wrapper(a.api.CreateTemplate))
		object.GET("", utils.Wrapper(a.api.ListTemplates))
		object.GET("/:name", utils.Wrapper(a.api.GetTemplate))
		object.PUT("/:name", utils.Wrapper(a.api.UpdateTemplate))
		object.DELETE("/:name", utils.Wrapper(a.api.DeleteTemplate))
	}
	{
		object := v1.Group("/clusters")
		object.POST("", utils.Wrapper(a.api.CreateCluster))
		object.GET("", utils.Wrapper(a.api.ListClusters))
		object.GET("/:name", utils.Wrapper(a.api.GetCluster))
		object.PUT("/:name", utils.Wrapper(a.api.UpdateCluster))
		object.DELETE("/:name", utils.Wrapper(a.api.DeleteCluster))
	}
}
