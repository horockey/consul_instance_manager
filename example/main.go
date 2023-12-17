package main

import (
	"context"
	"fmt"
	"time"

	go-consul-instance-manager "github.com/horockey/go-consul-instance-manager"
)

const serviceName = "my_awesome_service"

func main() {
	iman, err := go-consul-instance-manager.NewClient(
		serviceName,
		go-consul-instance-manager.WithPollInterval(time.Millisecond*500),
		go-consul-instance-manager.WithDownHoldDuration(time.Second*3),
	)
	panicOnErr(err)

	go iman.Start(context.TODO())

	go func() {
		time.Sleep(time.Second * 2)
		iman.Register("host1", "http://host1:8080")
		iman.Register("host2", "http://host2:8080")
		time.Sleep(time.Second * 2)
		iman.Deregister("host1")
	}()
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		inses, err := iman.GetInstances()
		panicOnErr(err)
		for _, ins := range inses {
			fmt.Printf("%s (%s)\n", ins.Name(), ins.Status().String())
		}
		fmt.Println()
	}
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}
