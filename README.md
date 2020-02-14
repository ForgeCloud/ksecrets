# ksecrets
A Kustomize Plugin to get Secrets out of various Secret Managers

Each Kubernetes secret object is represented by one object of kind `EncryptedSecret`.
The metadata.name and metadata.namespace of the object will be the name and namespace of
the Kubernetes secret, with a possible suffix hash. The key names are each represented by
a secret in a Secrets Manager, see below for naming.

## Currently Supported Secret Managers
* Google Secret Manager
* AWS Secrets Manager

## Currently Supported versions of Kustomize and Golang
* The Go plugin has to be built with the exact same version of Go modules as Kustomize.
* We intend to add more versions to the releases as Kustomize progresses.
* Kustomize versions
  * `v3.5.4`
* Go version 1.13 or higher

## Notes
* Kustomize updates all references to a secret's name in all other Kubernetes objects, even when a suffix hash is used.
* You can disable the suffix hash by setting `disableNameSuffixHash: true`, see [examples](example).
* You can set the Kubernetes secret `type` for TLS secrets and the like, see [examples](example).
* You can set the Kustomize `behavior:` to `replace`, `merge`, or `create` (default is `create`.)

## Naming Secrets Manager Secrets
* To name each Secret Manager secret
  * Use lower case in key names
  * Take the metadata.name of your `EncyptedSecret` and join it to the key name using an underscore
  * Replace all instances of `.` and `/` in key names with `_`
* Also see [examples](example)
* Example: an `EncryptedSecret` object named `my-secrets` with keys `creds.json` and
`ca.crt` translate to secrets in a Secrets Manager named `my-secrets_creds_json`
and `my-secrets_ca_crt` respectively, and the YAML file would look like this:

```yaml
apiVersion: crd.forgecloud.com/v1
kind: EncryptedSecret
metadata:
  name: my-secrets
  namespace: default
source: GCP
gcpProjectID: my-gcp-project-id
keys:
- creds.json
- ca.crt
```
These files are then referenced in a `kustomization.yaml` under `generators:`.

## Operating the plugin

### Authentication to a Secrets Manager
The plugin uses Go libraries provided by GCP and AWS, both of which automatically try various forms of authentication.

#### GCP Secret Manager
* Run [`gcloud auth application-default login`](https://cloud.google.com/sdk/gcloud/reference/auth/application-default/), follow the instructions, done, OR
* Set the `GOOGLE_APPLICATION_CREDENTIALS` environment variable to the path of a GCP Service or User Account credentials file.
* For additional options and more information, see the [library docs](https://pkg.go.dev/cloud.google.com/go@v0.53.0?tab=doc).

#### AWS Secrets Manager
* [Setup](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html) an `~/.aws/credentials` file, done, OR
* For additional options and more information, see "Configuring Credentials" [here](https://docs.aws.amazon.com/sdk-for-go/api/).

### Running Kustomize with the plugin
* Kustomize expects the Go plugin to be here: `$XDG_CONFIG_HOME/kustomize/plugin/crd.forgecloud.com/v1/encryptedsecret/EncryptedSecret.so`.
  * For more information, see [the docs](https://github.com/kubernetes-sigs/kustomize/tree/0075d0a88c380f22ed27c324cbeda05c0640e885/docs/plugins#placement).
* On most Unix systems, `$XDG_CONFIG_HOME` is `~/.config`, but you can set it to wherever you like. We often just build the plugin in place and set `XDG_CONFIG_HOME=.`, as seen below.
* Build and run the plugin without Docker like this:
```shell
git clone git@github.com:ForgeCloud/ksecrets.git
cd ksecrets
go get -gcflags="all=-N -l" sigs.k8s.io/kustomize/kustomize/v3@v3.5.4
cd kustomize/plugin/crd.forgecloud.com/v1/encryptedsecret
go build -gcflags="all=-N -l" -buildmode plugin -o EncryptedSecret.so encryptedsecret.go
cd ../../../../..
# assuming your Kustomize configs are in ./kustomize
XDG_CONFIG_HOME=. kustomize build --enable_alpha_plugins kustomize/overlays/my-overlay
```
* Build and run the plugin with Docker like this, swapping out your email address if using GCP:
```shell
git clone git@github.com:ForgeCloud/ksecrets.git
cd ksecrets
docker build -t my-image-repo:latest .
# assuming your Kustomize configs are in ./kustomize
# for GCP
docker run -it --rm -v $(pwd):/mnt \
    -v $HOME/.config/gcloud/legacy_credentials/myemail@example.com/adc.json:/credentials/adc.json \
    -e GOOGLE_APPLICATION_CREDENTIALS=/credentials/adc.json \
    my-image-repo:latest \
    -- 'kustomize build --enable_alpha_plugins /mnt/kustomize/overlays/my-overlay'
# for AWS
docker run -it --rm -v $(pwd):/mnt \
    -v $HOME/.aws/credentials:/root/.aws/credentials \
    my-image-repo:latest \
    -- 'kustomize build --enable_alpha_plugins /mnt/kustomize/overlays/my-overlay'
```

