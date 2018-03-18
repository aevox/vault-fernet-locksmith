package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	health "github.com/docker/go-healthcheck"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	consulapi "github.com/hashicorp/consul/api"

	"github.com/aevox/vault-fernet-locksmith/pkg/config"
	"github.com/aevox/vault-fernet-locksmith/pkg/consul"
	"github.com/aevox/vault-fernet-locksmith/pkg/locksmith"
	"github.com/aevox/vault-fernet-locksmith/pkg/vault"
)

var (
	cfg              config.Configuration
	locksmithVersion string
	lock             *consulapi.Lock
	lockCh           <-chan struct{}
)

func init() {
	config.DefineCmdFlags(&cfg)
}

func main() {
	if err := config.GetConfig(&cfg); err != nil {
		glog.Fatalf("Error getting configuration: %v", err)
	}

	if cfg.Version {
		fmt.Println(locksmithVersion)
		os.Exit(0)
	}

	glog.Info("Initializing...")
	//Create vault client
	vaultClient, err := vault.NewClient()
	glog.V(1).Info("Creating vault client")
	if err != nil {
		glog.Fatalf("Failed to create vault client: %v", err)
	}
	//Set vault client token
	glog.V(1).Info("setting vault token")

	var vaultToken string
	if cfg.PrimaryVault.Token != "" {
		glog.V(1).Info("Using vault token from command option")
		vaultToken = cfg.PrimaryVault.Token
	} else if cfg.PrimaryVault.TokenFile != "" {
		data, err := ioutil.ReadFile(cfg.PrimaryVault.TokenFile)
		if err != nil {
			glog.Fatalf("Cannot read token file: %v", err)
		}
		vaultToken = string(data)
	} else if e := os.Getenv("VAULT_TOKEN"); e != "" {
		glog.V(1).Info("Using vault token from environment")
		vaultToken = strings.TrimSpace(e)
	} else {
		glog.Fatalf("No vault token provided")
	}
	vaultClient.Client.SetToken(vaultToken)

	// Create goroutine to renew vault token
	if cfg.PrimaryVault.TokenRenew {
		vaultClient.RenewToken()
	}

	if cfg.Health {
		health.Register("vaultChecker", health.PeriodicThresholdChecker(vaultChecker(vaultClient, cfg.SecretPath), time.Second*15, 3))

		go func() {
			// create http server to expose health status
			r := mux.NewRouter()

			r.HandleFunc("/health", health.StatusHandler)

			srv := &http.Server{
				Handler:     r,
				Addr:        "0.0.0.0:8080",
				ReadTimeout: 15 * time.Second,
			}
			glog.Fatal(srv.ListenAndServe())
		}()
	}

	if cfg.Lock {
		glog.V(1).Info("Creating consul client")
		var consulToken string
		if cfg.ConsulToken != "" {
			consulToken = cfg.ConsulToken
		}
		consulClient, err := consul.NewClient(consulToken)
		if err != nil {
			glog.Fatalf("Failed to create consul client: %v", err)
		}
		if cfg.Health {
			health.Register("consulChecker", health.PeriodicThresholdChecker(consulChecker(consulClient.Client, cfg.LockKey), time.Second*15, 3))
		}
		glog.Info("Attempting to acquire lock")
		lock, err = consulClient.Client.LockKey(cfg.LockKey)
		if err != nil {
			glog.Fatalf("Lock setup failed :%v", err)
		}
		stopCh := make(chan struct{})
		lockCh, err = lock.Lock(stopCh)
		if err != nil {
			glog.Fatalf("Failed acquiring lock: %v", err)
		}
		glog.Info("Lock acquired")

		// Handle SIGINT and SIGTERM.
		sigs := make(chan os.Signal)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			for {
				select {
				case <-lockCh:
					glog.Fatalf("Lost lock, Exting")
					os.Exit(1)
				case sig := <-sigs:
					glog.Infof("Recieved signal: %v", sig)
					if cfg.Lock {
						// Attempt to release lock and destroy it
						if err := consul.CleanLock(lock); err != nil {
							glog.Errorf("Error cleaning consul lock: %v", err)
							os.Exit(1)
						}
					}
					os.Exit(0)
				}
			}
		}()
	}

	glog.Info("Initialization complete")
	glog.Info("Starting...")
	locksmith.Run(vaultClient, cfg.SecretPath, cfg.TTL)
}
