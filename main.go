package main

import (
	"fmt"

	"k8s.io/client-go/tools/portforward"
)

func main() {
	fmt.Println(portforward.ForwardedPort{})
}
