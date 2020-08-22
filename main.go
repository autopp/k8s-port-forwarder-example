package main

import (
	"net/http"
	"os"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type MyPortForwardOptions struct {
	configFlags *genericclioptions.ConfigFlags
}

func main() {
	transport, upgrader, err := spdy.RoundTripperFor(&rest.Config{})
	if err != nil {
		panic(err)
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", nil)
	fw, err := portforward.New(dialer, nil, nil, nil, os.Stdout, os.Stderr)

	if err != nil {
		panic(err)
	}

	fw.ForwardPorts()
}
