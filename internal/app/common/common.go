package common

import (
	"fmt"
	"os"
	"path"
	"plugin"

	"github.com/Rightech/ric-edge/pkg/jsonrpc"
)

func LoadPlugin(name string) (jsonrpc.Caller, error) {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	p, err := plugin.Open(path.Join(dir, "plugins", name, fmt.Sprintf("%s.so", name)))
	if err != nil {
		return nil, err
	}

	getFn, err := p.Lookup("GetCaller")
	if err != nil {
		panic(err)
	}

	cl := getFn.(func() jsonrpc.Caller)()

	return cl, nil
}
