### **Vault-Fernet-Locksmith**

Locksmith rotates periodically [Fernet Keys](https://github.com/fernet/spec) in a Hashicorp's [Vault](https://www.vaultproject.io).

Locksmith implements a lock feature using the [Consul](https://www.consul.io/) lock system to ensure that only one instance of this locksmith is running.

##### **Build**

A simple `make` will build the project.

##### **Usage**

```
Usage of ./bin/locksmith:
  -alsologtostderr
    	log to standard error as well as files
  -consul-creds string
    	Path to consul token in vault
  -consul-token string
    	Consul token used to authenticate with consul
  -health
    	enable endpoint /health on port 8080
  -lock
    	Acquires a lock with consul to ensure that only one instance of locksmith is running
  -lock-key string
    	Key used for locking (default "locks/locksmith/.lock")
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -renew-vault-token
    	Renew vault token
  -secret-path string
    	Path to the fernet-keys secret in vault (default "secret/fernet-keys")
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -ttl int
    	Interval between each vault secret fetch (default 120)
  -v value
    	log level for V logs
  -vault-token string
    	Vault token used to authenticate with vault
  -vault-token-file string
    	File containing the vault token used to authenticate with vault
  -version
    	Prints locksmith version, exits
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```

##### **Bootstrap**

Fernet keys are stored in vault as a single secret. (default `secret/fernet-keys`)

```yaml
creation_time: 1516626452
keys:
- -_Ljq7IAx57gtPPuZloOKRpt_4LoIZ54awQs6-vzRXs=
- awYgumbNGJpu5sj1adgbVPLVOAey6o5qlPvaJ8c-DRQ=
- dvhnpz2MlYwLWbZgueFSjuuecTbCvOF8siKGQVAjVno=
period: 3600
ttl: 120s
```

Vault-Fernet-Locksmith does not create a fernet keys secret if it is not present in Vault. This is intended not to interfere with the keys rotation if the secret in vault is corrupted or deleted.
The number of fernet keys and the rotation period are set at bootstrap. You have to manually edit them if you want to change them afterwards.

You can use the locksmith-bootsrap binary to write the first secret to vault.


##### **Configuration**

|  command line option |    environment option   |        default value       |
|----------------------|-------------------------|----------------------------|
| `-vault-token`       | `VAULT_TOKEN`           | `""`                       |
| `-renew-vault-token` | `VFL_RENEW_VAULT_TOKEN` | `false`                    |
| `-secret-path`       | `VFL_SECRET_PATH`       | `"secret/fernet-keys"`     |
| `-ttl`               | `VFL_TTL`               | `120`                      |
| `-health`            | `VFL_HEALTH`            | `false`                    |
| `-lock`              | `VFL_LOCK`              | `false`                    |
| `-lock-key`          | `VFL_LOCK_KEY`          | `"locks/locksmith/.lock"`  |
| `-consul-creds`      | `VFL_CONSUL_CREDS`      | `""`                       |
| `-consul-token`      | `VFL_CONSUL_TOKEN`      | `""`                       |


##### **TODO**

- Tests
