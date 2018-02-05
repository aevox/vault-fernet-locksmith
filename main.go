package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/pkg/flagutil"
	health "github.com/docker/go-healthcheck"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	consulapi "github.com/hashicorp/consul/api"

	"github.com/aevox/vault-fernet-locksmith/consul"
	"github.com/aevox/vault-fernet-locksmith/locksmith"
	"github.com/aevox/vault-fernet-locksmith/vault"
)

var (
	options          Options
	locksmithVersion string
	lock             *consulapi.Lock
	lockCh           <-chan struct{}
)

// Options is all the different options you can pass to locksmith
type Options struct {
	version         bool
	lock            bool
	lockKey         string
	health          bool
	consulToken     string
	consulCreds     string
	vaultToken      string
	vaultTokenFile  string
	vaultTokenRenew bool
	secretPath      string
	ttl             int
}

func init() {
	flag.BoolVar(&options.version, "version", false, "Prints locksmith version, exits")
	flag.BoolVar(&options.lock, "lock", false, "Acquires a lock with consul to ensure that only one instance of locksmith is running")
	flag.StringVar(&options.lockKey, "lock-key", "locks/locksmith/.lock", "Key used for locking")
	flag.BoolVar(&options.health, "health", false, "enable endpoint /health on port 8080")
	flag.StringVar(&options.vaultToken, "vault-token", "", "Vault token used to authenticate with vault")
	flag.StringVar(&options.vaultTokenFile, "vault-token-file", "", "File containing the vault token used to authenticate with vault")
	flag.BoolVar(&options.vaultTokenRenew, "renew-vault-token", false, "Renew vault token")
	flag.StringVar(&options.consulCreds, "consul-creds", "", "Path to consul token in vault")
	flag.StringVar(&options.consulToken, "consul-token", "", "Consul token used to authenticate with consul")
	flag.StringVar(&options.secretPath, "secret-path", "secret/fernet-keys", "Path to the fernet-keys secret in vault")
	flag.IntVar(&options.ttl, "ttl", 120, "Interval between each vault secret fetch")
}

func main() {
	flag.Parse()
	glog.V(2).Infof("Options: %v", options)
	err := flagutil.SetFlagsFromEnv(flag.CommandLine, "VFL")
	if err != nil {
		glog.Fatalf("Cannot set flags from env: %v", err)
	}

	if options.version {
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
	if options.vaultToken != "" {
		glog.V(1).Info("Using vault token from command option")
		vaultToken = options.vaultToken
	} else if options.vaultTokenFile != "" {
		data, err := ioutil.ReadFile(options.vaultTokenFile)
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
	if options.vaultTokenRenew {
		go vaultClient.RenewToken()
	}

	if options.lock {
		glog.V(1).Info("Creating consul client")
		var consulToken string
		if options.consulToken != "" {
			glog.V(1).Info("Using consul token from option line")
			consulToken = options.consulToken
		} else if options.consulCreds != "" {
			glog.V(1).Infof("Using consul credentials from vault: %v\n", options.consulCreds)
			consulTokenSecret, err := vaultClient.ReadSecret(options.consulCreds)
			if err != nil {
				glog.Fatalf("Could not get consul token from vault: %v", err)
			}
			if consulTokenSecret.Data["token"] == nil {
				glog.Fatalf("Consul token in vault secret not found")
			}
			if str, ok := (consulTokenSecret.Data["token"]).(string); ok {
				consulToken = str
			} else {
				glog.Fatalf("Error converting token to string")
			}
			if consulTokenSecret.Renewable {
				vaultClient.Renewer(consulTokenSecret)
			}
		}
		consulClient, err := consul.NewClient(consulToken)
		if err != nil {
			glog.Fatalf("Failed to create consul client: %v", err)
		}
		if options.health {
			health.Register("consulChecker", health.PeriodicThresholdChecker(consulChecker(consulClient.Client, options.lockKey), time.Second*15, 3))
		}
		glog.Info("Attempting to acquire lock")
		lock, err = consulClient.Client.LockKey(options.lockKey)
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
					if options.lock {
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

	if options.health {
		health.Register("vaultChecker", health.PeriodicThresholdChecker(vaultChecker(vaultClient, options.secretPath), time.Second*15, 3))

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
	glog.Info("Initialization complete")
	glog.Info("Starting...")
	locksmith.Run(vaultClient, options.secretPath, options.ttl)
}
