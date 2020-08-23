package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type MyPortForwardOptions struct {
	configFlags *genericclioptions.ConfigFlags
	genericclioptions.IOStreams
}

func NewOptions() *MyPortForwardOptions {
	return &MyPortForwardOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
		IOStreams:   genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr},
	}
}

func NewCmd() *cobra.Command {
	o := NewOptions()
	cmd := &cobra.Command{
		Use:          "port-forward-do pod localport remoteport cmd...",
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			podName := args[0]
			localPort := args[1]
			remotePort := args[2]

			config, err := o.configFlags.ToRESTConfig()
			if err != nil {
				return err
			}
			config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
			if config.APIPath == "" {
				config.APIPath = "/api"
			}
			if config.NegotiatedSerializer == nil {
				// This codec factory ensures the resources are not converted. Therefore, resources
				// will not be round-tripped through internal versions. Defaulting does not happen
				// on the client.
				config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
			}

			transport, upgrader, err := spdy.RoundTripperFor(config)
			if err != nil {
				return err
			}
			namespace, _, err := o.configFlags.ToRawKubeConfigLoader().Namespace()
			if err != nil {
				return nil
			}

			builder := resource.NewBuilder(o.configFlags).
				WithScheme(scheme.Scheme, scheme.Scheme.PrioritizedVersionsAllGroups()...).
				ContinueOnError().
				NamespaceParam(namespace).DefaultNamespace()

			builder.ResourceNames("pods", podName)
			obj, err := builder.Do().Object()
			if err != nil {
				return err
			}

			pod, ok := obj.(*v1.Pod)
			if !ok {
				return fmt.Errorf("expected *v1.Pod, but got %T: %#v", obj, obj)
			}

			if pod.Status.Phase != v1.PodRunning {
				return fmt.Errorf("unnable to forward port because pod is not running. (status = %v)", pod.Status.Phase)
			}

			// Get RESTClient
			restClient, err := rest.RESTClientFor(config)
			if err != nil {
				return err
			}

			req := restClient.Post().
				Resource("pods").Namespace(namespace).Name(pod.Name).SubResource("portforward")

			dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", req.URL())
			ports := []string{fmt.Sprintf("%s:%s", localPort, remotePort)}
			stopChan := make(chan struct{}, 1)
			readyChan := make(chan struct{})

			fw, err := portforward.New(dialer, ports, stopChan, readyChan, os.Stdout, os.Stderr)

			if err != nil {
				return err
			}

			return fw.ForwardPorts()
		},
	}

	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

func main() {
	flags := pflag.NewFlagSet("port-forward-do", pflag.ExitOnError)
	pflag.CommandLine = flags
	root := NewCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
