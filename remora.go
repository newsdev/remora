package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buth/remora/vendor/src/github.com/coreos/go-etcd/etcd"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	etcdPeers, dockerEndpoint, hostIP string
	containerPort                     int64
	interval, buffer                  time.Duration
	useJSON                           bool
)

func init() {
	flag.StringVar(&etcdPeers, "C", "http://127.0.0.1:4001", "a comma-delimited list of machine addresses in the etcd cluster")
	flag.StringVar(&dockerEndpoint, "H", "unix:///var/run/docker.sock", "connection string for the Docker daemon")
	flag.StringVar(&hostIP, "a", "127.0.0.1", "host IP address")
	flag.Int64Var(&containerPort, "p", 80, "container port to report")
	flag.DurationVar(&interval, "i", 30*time.Second, "interval length")
	flag.DurationVar(&interval, "b", 5*time.Second, "buffer length")
	flag.BoolVar(&useJSON, "j", false, "set values in etcd as JSON")
}

type jsonValue struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
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
		keyFile := filepath.Join(dockerCertPath, "key.pem")
		certFile := filepath.Join(dockerCertPath, "cert.pem")
		caFile := filepath.Join(dockerCertPath, "ca.pem")
		dockerClient, err = docker.NewTLSClient(dockerEndpoint, certFile, keyFile, caFile)
		if err != nil {
			log.Fatalf("error: %s", err.Error())
		}
	}

	// Setup a new etcd client.
	etcdClient := etcd.NewClient(strings.Split(etcdPeers, ","))

	// containerName := fmt.Sprintf("/%s", flag.Arg(0))
	etcdKey := flag.Arg(1)

	// The loop.
	for sleep := int64(0); ; sleep = rand.Int63n(interval.Nanoseconds()) {
		log.Printf("sleeping %d nanoseconds", sleep)
		time.Sleep(time.Duration(sleep) * time.Nanosecond)

		container, err := dockerClient.InspectContainer(flag.Arg(0))
		if err != nil {
			log.Printf("docker error: %s", err.Error())
			continue
		}

		for _, portMapping := range container.NetworkSettings.PortMappingAPI() {

			// Check if this is the port that we are supposed to be tracking.
			// Otherwise, keep looking.
			if portMapping.PrivatePort == containerPort {

				// Set a value to save in etcd based on weather or not we are supposed
				// to encode a JSON value or not.
				var value string
				if useJSON {
					valueBytes, err := json.Marshal(jsonValue{Host: hostIP, Port: portMapping.PublicPort})
					if err != nil {
						log.Printf("json encoding error: %s", err.Error())
						break
					}

					value = string(valueBytes)
				} else {
					value = fmt.Sprintf("%s:%d", hostIP, portMapping.PublicPort)
				}

				// Try to sync the cluster before writing.
				etcdClient.SyncCluster()

				// Log the value and set it in etcd.
				log.Printf("setting value `%s`", value)
				if _, err := etcdClient.Set(etcdKey, value, uint64(interval.Seconds()+buffer.Seconds())); err != nil {
					log.Printf("etcd error: %s", err.Error())
					break
				}

				// We can quit the loop as we've successfully found what we were
				// looking for.
				break
			}
		}
	}
}
