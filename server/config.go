package server

import (
	"time"

	"github.com/baetyl/baetyl-controller/utils"
)

type ServerConfig struct {
	Server Server `yaml:"server" json:"server"`
}

// Server server config
type Server struct {
	Port         string            `yaml:"port" json:"port"`
	ReadTimeout  time.Duration     `yaml:"readTimeout" json:"readTimeout" default:"30s"`
	WriteTimeout time.Duration     `yaml:"writeTimeout" json:"writeTimeout" default:"30s"`
	ShutdownTime time.Duration     `yaml:"shutdownTime" json:"shutdownTime" default:"3s"`
	Certificate  utils.Certificate `yaml:",inline" json:",inline"`
}
