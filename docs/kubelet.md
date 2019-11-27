# kubelet hacking

knode can be used to enable 
[Dynamic Kubelet Configuration](https://kubernetes.io/blog/2018/07/11/dynamic-kubelet-configuration/).

#### Installing knode

knode should be installed with defaults first. Once the knode daemonset is installed, applying the
update to enable dynamic kubelet configuration will be controlled by rolling update. It's currently
configured to update a node at a time, with two minutes in between nodes.

``` bash
# Install knode with defaults
curl -L https://github.com/juan-lee/knode/releases/download/v0.1.1/knode-default.yaml | kubectl apply -f -
kubectl rollout status daemonset -n knode-system knode-daemon

# Update knode to move /var/lib/docker to /mnt/docker
curl -L https://github.com/juan-lee/knode/releases/download/v0.1.1/knode-kubelet.yaml | kubectl apply -f -
kubectl rollout status daemonset -n knode-system knode-daemon
```

# Reconfigure a Node's Kubelet to use Dynamic Configuration

Follow [these](https://kubernetes.io/docs/tasks/administer-cluster/reconfigure-kubelet/)
instructions.


