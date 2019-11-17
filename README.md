# knode daemon

knode uses a kubernetes daemonset for node configuration.

## What's configurable?

### /etc/docker/daemon.json

The primary purpose for knode is to be able to move docker's `data-root` from `/var/lib/docker` to
`/mnt/docker`.

#### Installing knode

knode should be installed with defaults first. Once the knode daemonset is installed, applying the
update to move the docker `data-root` will be controlled by rolling update. It's currently
configured to update a node at a time, with two minutes in between nodes.

``` bash
# Install knode with defaults
curl -L https://github.com/juan-lee/knode/releases/download/v0.1.0/knode-default.yaml | kubectl apply -f -
kubectl rollout status daemonset -n knode-system knode-daemon

# Update knode to move /var/lib/docker to /mnt/docker
curl -L https://github.com/juan-lee/knode/releases/download/v0.1.0/knode-tmpdir.yaml | kubectl apply -f -
kubectl rollout status daemonset -n knode-system knode-daemon
```
