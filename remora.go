package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	etcdPeers, dockerEndpoint, hostIP, containerPort string
	interval                                         time.Duration
)

func init() {
	flag.StringVar(&etcdPeers, "C", "http://127.0.0.1:4001", "a comma-delimited list of machine addresses in the etcd cluster")
	flag.StringVar(&dockerEndpoint, "H", "unix:///var/run/docker.sock", "connection string for the Docker daemon")
	flag.StringVar(&hostIP, "a", "127.0.0.1", "host IP address")
	flag.StringVar(&containerPort, "p", "80", "container port to report")
	flag.DurationVar(&interval, "i", time.Minute, "interval length")
}

func main() {
	flag.Parse()

	var (
		dockerClient *docker.Client
		err          error
	)

	if dockerCertPath := os.Getenv("DOCKER_CERT_PATH"); dockerCertPath == "" {
		dockerClient, err = docker.NewClient(dockerEndpoint)
		if err != nil {
			log.Fatalf("error: %s", err.Error())
		}
	} else {
		key := filepath.Join(dockerCertPath, "key.pem")
		cert := filepath.Join(dockerCertPath, "cert.pem")
		ca := filepath.Join(dockerCertPath, "ca.pem")
		dockerClient, err = docker.NewTLSClient(dockerEndpoint, cert, key, ca)
		if err != nil {
			log.Fatalf("error: %s", err.Error())
		}
	}

	// Setup a new etcd client.
	etcdClient := etcd.NewClient(strings.Split(etcdPeers, ","))
	fmt.Println(etcdClient)

	containerID := flag.Arg(0)
	etcdKey := flag.Arg(1)
	n := interval.Nanoseconds()
	nd2 := n / 2

	// The loop.
	for {

		sleep := nd2 + rand.Int63n(nd2)
		log.Printf("sleeping %d nanoseconds", sleep)
		time.Sleep(time.Duration(sleep) * time.Nanosecond)

		// Inspect the the given container. The relevant port-bindings should be
		// present in the resulting container definition.
		container, err := dockerClient.InspectContainer(containerID)
		if err != nil {
			log.Printf("error: %s", err.Error())
			continue
		}

		// Attempt to find a port-binding for the given container port.
		var containerPortBinding string
		for port, portBinding := range container.NetworkSettings.Ports {
			if port.Port() == containerPort {
				containerPortBinding = portBinding[0].HostPort
			}
		}

		// Insure the port number can be converted to a number.
		port, err := strconv.Atoi(containerPortBinding)
		if err != nil {
			log.Printf("error: %s", err.Error())
			continue
		}

		// Insure the port number is valid.
		if port <= 0 {
			log.Printf("error: invalid port %d from parsed string \"%s\"", port, containerPortBinding)
			continue
		}

		// Save the value in etcd.
		etcdClient.SyncCluster()
		log.Println(etcdClient.GetCluster())
		value := fmt.Sprintf("%s:%s", hostIP, containerPortBinding)
		if _, err := etcdClient.Set(etcdKey, value, uint64(interval.Seconds())+1); err != nil {
			log.Printf("error: %s", err.Error())
		}
	}
}
