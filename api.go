package cni

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/appc/cni/pkg/plugin"
)

type RuntimeConf struct {
	ContainerID string
	NetNS       string
	IfName      string
	Args        string
}

type CNI interface {
	AddNetwork(net *plugin.NetConf, rt *RuntimeConf) (*plugin.Result, error)
	DelNetwork(net *plugin.NetConf, rt *RuntimeConf) error
}

type CNIConfig struct {
	Path []string
}

func (c *CNIConfig) AddNetwork(net *plugin.NetConf, rt *RuntimeConf) (*plugin.Result, error) {
	retBytes, err := c.execPlugin("ADD", net, rt)
	if err != nil {
		return nil, err
	}
	var res plugin.Result
	if err = json.Unmarshal(retBytes, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *CNIConfig) DelNetwork(net *plugin.NetConf, rt *RuntimeConf) error {
	return nil
}

// =====

// taken from rkt/networking/podenv.go
// also note similar code in cni/pkg/plugin/ipam.go
func (c *CNIConfig) findPlugin(plugin string) string {
	for _, p := range c.Path {
		fullname := filepath.Join(p, plugin)
		if fi, err := os.Stat(fullname); err == nil && fi.Mode().IsRegular() {
			return fullname
		}
	}

	return ""
}

// taken from rkt/networking/net_plugin.go
func (c *CNIConfig) execPlugin(action string, conf *plugin.NetConf, rt *RuntimeConf) ([]byte, error) {
	pluginPath := c.findPlugin(conf.Type)
	if pluginPath == "" {
		return nil, fmt.Errorf("could not find plugin %q in %v", conf.Type, c.Path)
	}

	vars := [][2]string{
		{"CNI_COMMAND", action},
		{"CNI_CONTAINERID", rt.ContainerID},
		{"CNI_NETNS", rt.NetNS},
		{"CNI_ARGS", rt.Args},
		{"CNI_IFNAME", rt.IfName},
		{"CNI_PATH", strings.Join(c.Path, ":")},
	}

	confBytes, err := json.Marshal(conf)
	if err != nil {
		return nil, err
	}
	stdin := bytes.NewBuffer(confBytes)
	stdout := &bytes.Buffer{}

	cmd := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Env:    envVars(vars),
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: os.Stderr,
	}

	err = cmd.Run()
	return stdout.Bytes(), err
}

// taken from rkt/networking/net_plugin.go
func envVars(vars [][2]string) []string {
	env := os.Environ()

	for _, kv := range vars {
		env = append(env, strings.Join(kv[:], "="))
	}

	return env
}
