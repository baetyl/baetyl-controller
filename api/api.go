package api

import (
	"errors"

	"github.com/baetyl/baetyl-controller/kube"
)

var (
	ErrNotFound = errors.New("resource not found")
)

type AdminAPI struct {
	cli kube.Client
}

func NewAdminAPI(cli kube.Client)(*AdminAPI,error){
	return &AdminAPI{cli: cli},nil
}




