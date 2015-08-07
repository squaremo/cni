package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/appc/cni"
	"github.com/appc/cni/pkg/plugin"
)

const (
	EnvCNIPath = "CNI_PATH"
	EnvNetDir  = "NETCONFPATH"

	DefaultNetDir = "/etc/cni/net.d"

	CmdAdd = "add"
	CmdDel = "del"
)

func loadNetConf(dir, name string) (*plugin.NetConf, error) {
	filename := fmt.Sprintf("%s.conf", name)
	file := filepath.Join(dir, filename)
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var conf plugin.NetConf
	if err = json.Unmarshal(bytes, &conf); err != nil {
		return nil, err
	}
	return &conf, nil
}

func main() {
	if len(os.Args) < 3 {
		usage()
		return
	}

	netdir := os.Getenv(EnvNetDir)
	if netdir == "" {
		netdir = DefaultNetDir
	}
	netconf, err := loadNetConf(netdir, os.Args[2])
	if err != nil {
		exit(err)
	}

	netns := os.Args[3]

	cninet := cni.FromConfig(&cni.CNIConfig{
		Path: strings.Split(os.Getenv(EnvCNIPath), ":"),
	})

	rt := &cni.RuntimeConf{
		ContainerID: "cni",
		NetNS:       netns,
		IfName:      "eth0",
		Args:        "",
	}

	switch os.Args[1] {
	case CmdAdd:
		_, err := cninet.AddNetwork(netconf, rt)
		exit(err)
	case CmdDel:
		exit(cninet.DelNetwork(netconf, rt))
	}
}

func usage() {
	exe := filepath.Base(os.Args[0])

	fmt.Fprintf(os.Stderr, "%s: Add or remove network interfaces from a network namespace\n", exe)
	fmt.Fprintf(os.Stderr, "  %s %s <net> <netns>\n", exe, CmdAdd)
	fmt.Fprintf(os.Stderr, "  %s %s <net> <netns>\n", exe, CmdDel)
	os.Exit(1)
}

func exit(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
