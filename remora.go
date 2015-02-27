package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	etcdPeers, dockerEndpoint, hostIP string
	containerPort                     int64
	interval                          time.Duration
)

func init() {
	flag.StringVar(&etcdPeers, "C", "http://127.0.0.1:4001", "a comma-delimited list of machine addresses in the etcd cluster")
	flag.StringVar(&dockerEndpoint, "H", "unix:///var/run/docker.sock", "connection string for the Docker daemon")
	flag.StringVar(&hostIP, "a", "127.0.0.1", "host IP address")
	flag.Int64Var(&containerPort, "p", 80, "container port to report")
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

	containerName := fmt.Sprintf("/%s", flag.Arg(0))
	etcdKey := flag.Arg(1)
	n := interval.Nanoseconds()
	nd2 := n / 2

	// The loop.
	for {

		sleep := nd2 + rand.Int63n(nd2)
		log.Printf("sleeping %d nanoseconds", sleep)
		time.Sleep(time.Duration(sleep) * time.Nanosecond)

		containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: true})
		if err != nil {
			log.Printf("error: %s", err.Error())
			continue
		}

		for _, container := range containers {
			for _, name := range container.Names {
				if name == containerName {
					for _, port := range container.Ports {
						if containerPort == port.PrivatePort {
							value := fmt.Sprintf("%s:%d", hostIP, port.PublicPort)
							log.Printf("setting %s", value)

							dirs := strings.Split(etcdKey, "/")
							for i := 2; i < len(dirs); i++ {
								if _, err := etcdClient.RawSetDir(strings.Join(dirs[0:i], "/"), 0); err != nil {
									log.Printf("error: %s", err.Error())
									continue
								}
							}

							if _, err := etcdClient.Set(etcdKey, value, uint64(interval.Seconds())+1); err != nil {
								log.Printf("error: %s", err.Error())
							}
						}
					}
				}
			}
		}
	}
}
