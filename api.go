package cni

import (
	"github.com/appc/cni/pkg/plugin"
)

type RuntimeConf struct {
	ContainerID string
	NetNS       string
	IfName      string
	Args        []string
}

type CNI interface {
	AddNetwork(net *plugin.NetConf, rt *RuntimeConf) (*plugin.Result, error)
	DelNetwork(net *plugin.NetConf, rt *RuntimeConf) error
}

type CNIConfig struct {
	Path []string
}

func LoadNetConf(dir, name string) (*plugin.NetConf, error) {
	return nil, nil
}

func FromConfig(conf *CNIConfig) CNI {
	return conf
}

func (c *CNIConfig) AddNetwork(net *plugin.NetConf, rt *RuntimeConf) (*plugin.Result, error) {
	return nil, nil
}

func (c *CNIConfig) DelNetwork(net *plugin.NetConf, rt *RuntimeConf) error {
	return nil
}
