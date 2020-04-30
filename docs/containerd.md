## containerd CRI

knode can be used to replace moby with containerd for the node's container runtime.


### Installing knode

knode should be installed with defaults first. Once the knode daemonset is installed, applying the
update to configure containerd will be controlled by rolling update. It's currently configured to
update a node at a time, with two minutes in between nodes.

``` bash
# Install ip-masq-agent
curl -L https://raw.githubusercontent.com/kubernetes-sigs/ip-masq-agent/master/ip-masq-agent.yaml | kubectl apply -f -

# Install knode with defaults
curl -L https://github.com/juan-lee/knode/releases/download/v0.1.3/knode-default.yaml | kubectl apply -f -
kubectl rollout status daemonset -n knode-system knode-daemon

# Update knode to reconfigure the node to use containerd
curl -L https://github.com/juan-lee/knode/releases/download/v0.1.3/knode-containerd.yaml | kubectl apply -f -
kubectl rollout status daemonset -n knode-system knode-daemon
```
