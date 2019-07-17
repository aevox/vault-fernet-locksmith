### **Vault-Fernet-Locksmith**

Locksmith periodically rotates [Fernet Keys](https://github.com/fernet/spec) in Hashicorp's [Vault(s)](https://www.vaultproject.io).
It is intended to link [keystone](https://docs.openstack.org/keystone/latest/) ([openstack](https://www.openstack.org/)) and Vault for fernet keys management.

Locksmith implements a lock feature using [Consul](https://www.consul.io/) to make sure that only one instance of locksmith is running.


```
Usage:
  vault-fernet-locksmith [command]

Available Commands:
  bootstrap   Generate first set of fernet keys in Vault(s)
  delete      Delete fernet keys secret in Vault(s)
  help        Help about any command
  print       Print secrets stored in Vault(s)
  rotate      Force a fernet keys rotation
  version     Print version and exit
  watch       Watch keys in Vault(s) and rotate them when needed

Flags:
  -c, --config string             configuration file
  -h, --help                      help for vault-fernet-locksmith
      --secret-path string        path to the fernet-keys secret in primary Vault (default "secret/fernet-keys")
      --vault-address string      Vault address (default "https://127.0.0.1:8500")
      --vault-proxy string        proxy URL used to contact Vault
      --vault-token string        Vault token used to authenticate with Vault
      --vault-token-file string   file containing the vault token used to authenticate with Vault
  -v, --verbosity string          log level (debug, info, warn, error, fatal, panic) (default "info")

Use "vault-fernet-locksmith [command] --help" for more information about a command.
```

##### **Bootstrap**

Fernet keys are stored in vault as a single secret (default `secret/fernet-keys`).

```yaml
creation_time: 1516626452
keys:
- -_Ljq7IAx57gtPPuZloOKRpt_4LoIZ54awQs6-vzRXs=
- awYgumbNGJpu5sj1adgbVPLVOAey6o5qlPvaJ8c-DRQ=
- dvhnpz2MlYwLWbZgueFSjuuecTbCvOF8siKGQVAjVno=
period: 3600
ttl: 120s
```

You can use the bootstrap command to write the first secret to the Vault(s).

##### **Configuration**

vault-fernet-locksmith accepts a yaml or json configuration file (See [config.example.yaml](config.example.yaml)).

For minimal configuration you can set flags or environment variables:

|  command line option  |    environment variable       |        default value       |
|-----------------------|-------------------------------|----------------------------|
| `--vault-address`     | `VFL_VAULT_ADDRESS`           | `""`                       |
| `--vault-proxy`       | `VFL_VAULT_PROXY`             | `""`                       |
| `--vault-token`       | `VFL_VAULT_VAULT_TOKEN`       | `""`                       |
| `--vault-token-file`  | `VFL_VAULT_TOKEN_FILE`        | `""`                       |
| `--secret-path`       | `VFL_SECRETPATH`              | `"secret/fernet-keys"`     |
| `--ttl`               | `VFL_TTL`                     | `120`                      |
| `--health`            | `VFL_HEALTH`                  | `false`                    |
| `--health-period`     | `VFL_HEALTHPERIOD`            | `120`                      |
| `--consul-address`    | `VFL_CONSUL_ADDRESS`          | `""`                       |
| `--consul-proxy`      | `VFL_CONSUL_PROXY`            | `""`                       |
| `--consul-token`      | `VFL_CONSUL_TOKEN`            | `""`                       |
| `--consul-token-file` | `VFL_CONSUL TOKENFILE`        | `""`                       |
| `--lock`              | `VFL_CONSUL_LOCK`             | `false`                    |
| `--lock-key`          | `VFL_CONSUL_LOCKKEY`          | `"locks/locksmith/.lock"`  |
| `--verbosity`         | `VFL_VERBOSITY`               | `"info"`                   |

##### **Build**

A simple `make` will build the project.
