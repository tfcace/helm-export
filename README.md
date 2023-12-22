# helm-export

## What is it
Helm v3 stored all of it's state, alongside the rendered manifests inside of a single Secret (a secret per installation revision).  
`helm-export` is a tiny CLI utlity that exports or dumps the Kubernetes manifests that are deployed as part of a specific installation of Helm release.

### Usage
The basic usage is as follows:
```bash
helm-export <secret-name> [output-dir] [options]
```
Where:
* `secret-name` is the Kuberentes secret name, and it is manadatory.
* `output-dir` is a path to an existing folder in which the resulting manifest files will be stored.  
  If unspecified, the current working directory will be used.

The following options are also optional:
* `-n` for specificying the secret namespace
* `-kubeconfig` for secifying the path to `kubeconfig` (will default to `$HOME/.kube/config`)

## Motivation
Rather hackish to say the least.
There were times where I needed to to manipulate a manifest (or restore a deleted one), without having to go through the trouble of actually using Helm.


## Credits
[This](https://dbafromthecold.com/2020/08/10/decoding-helm-secrets/) is the tutorial I initially used to understand how Helm stores data in its serets, and fix some manifests by hand.

This [podinfo Chart](https://artifacthub.io/packages/helm/podinfo/podinfo) was super helpful for setting up a basic Helm release that I can base my tests on something concrete.  
Thanks @stefanprodan
