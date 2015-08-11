package cni

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type NetworkConfig struct {
	plugin.NetConf
	Bytes []byte
}

func ConfFromFile(filename string) (*NetworkConfig, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %s", filename, err)
	}
	conf := &NetworkConfig{Bytes: bytes}
	if err = json.Unmarshal(bytes, conf); err != nil {
		return nil, fmt.Errorf("error parsing %s: %s", filename, err)
	}
	return conf, nil
}

type CNI interface {
	AddNetwork(net *NetworkConfig, rt *RuntimeConf) (*plugin.Result, error)
	DelNetwork(net *NetworkConfig, rt *RuntimeConf) error
}

type CNIConfig struct {
	Path []string
}

func (c *CNIConfig) AddNetwork(net *NetworkConfig, rt *RuntimeConf) (*plugin.Result, error) {
	return c.execPlugin("ADD", net, rt)
}

func (c *CNIConfig) DelNetwork(net *NetworkConfig, rt *RuntimeConf) error {
	_, err := c.execPlugin("DEL", net, rt)
	return err
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

// there's another in cni/pkg/plugin/ipam.go, but it assumes the
// environment variables are inherited from the current process
func (c *CNIConfig) execPlugin(action string, conf *NetworkConfig, rt *RuntimeConf) (*plugin.Result, error) {
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

	stdin := bytes.NewBuffer(conf.Bytes)
	stdout := &bytes.Buffer{}

	cmd := exec.Cmd{
		Path:   pluginPath,
		Args:   []string{pluginPath},
		Env:    envVars(vars),
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: os.Stderr,
	}

	if err := cmd.Run(); err != nil {
		return nil, pluginErr(err, stdout.Bytes())
	}

	res := &plugin.Result{}
	err := json.Unmarshal(stdout.Bytes(), res)
	return res, err
}

// taken from cni/pkg/plugin/ipam.go
func pluginErr(err error, output []byte) error {
	if _, ok := err.(*exec.ExitError); ok {
		emsg := plugin.Error{}
		if perr := json.Unmarshal(output, &emsg); perr != nil {
			return fmt.Errorf("netplugin failed but error parsing its diagnostic message %q: %v", string(output), perr)
		}
		details := ""
		if emsg.Details != "" {
			details = fmt.Sprintf("; %v", emsg.Details)
		}
		return fmt.Errorf("%v%v", emsg.Msg, details)
	}

	return err
}

// taken from rkt/networking/net_plugin.go
func envVars(vars [][2]string) []string {
	env := os.Environ()

	for _, kv := range vars {
		env = append(env, strings.Join(kv[:], "="))
	}

	return env
}
