/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"errors"
	"flag"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"time"

	"k8s.io/klog/v2"
)

func readFile(name string) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil && os.IsNotExist(err) {
		return []byte(""), nil
	} else if err != nil {
		return nil, err
	}
	defer file.Close()
	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func nsEnterCommand(arg ...string) error {
	args := append([]string{"-m/proc/1/ns/mnt", "--"}, arg...)
	if out, err := exec.Command("/usr/bin/nsenter", args...).CombinedOutput(); err != nil {
		return errors.New(string(out))
	}
	return nil
}

func daemonReload() error {
	return nsEnterCommand("/bin/systemctl", "daemon-reload")
}

func restartDocker() error {
	return nsEnterCommand("/bin/systemctl", "restart", "docker")
}

func restartKubelet() error {
	return nsEnterCommand("/bin/systemctl", "restart", "kubelet")
}

func restartContainerd() error {
	return nsEnterCommand("/bin/systemctl", "restart", "containerd")
}

func enableContainerd() error {
	return nsEnterCommand("/bin/systemctl", "enable", "containerd")
}

func updateContainerd() error {
	if err := nsEnterCommand(
		"/usr/bin/curl", "-s", "-L",
		"https://github.com/containerd/containerd/releases/download/v1.3.0/containerd-1.3.0.linux-amd64.tar.gz",
		"-o", "/tmp/containerd-1.3.0.linux-amd64.tar.gz",
	); err != nil {
		return err
	}
	return nsEnterCommand("/bin/tar", "xvzf", "/tmp/containerd-1.3.0.linux-amd64.tar.gz", "-C", "/usr")
}

func reboot() error {
	return nsEnterCommand("/bin/systemctl", "reboot")
}

func replaceIfChanged(src, dst string) (bool, error) {
	_, err := os.Stat(src)
	if os.IsNotExist(err) {
		return false, nil
	}

	current, err := readFile(dst)
	if err != nil {
		return false, err
	}
	desired, err := readFile(src)
	if err != nil {
		return false, err
	}

	if bytes.Equal(current, desired) {
		klog.Infof("%s already configured", dst)
		return false, nil
	}

	klog.Infof("Updating %s:\n%s", dst, desired)
	if err := ioutil.WriteFile(dst, desired, 0644); err != nil {
		return false, err
	}
	return true, nil
}

func configureDockerDaemon() error {
	changed, err := replaceIfChanged("/configs/daemon.json", "/etc/docker/daemon.json")
	if err != nil {
		return err
	}
	if changed {
		if err := restartDocker(); err != nil {
			return err
		}
		if err := restartKubelet(); err != nil {
			return err
		}
	}
	return nil
}

func configureRuntimeSlice() (bool, error) {
	return replaceIfChanged("/configs/runtime.slice", "/etc/systemd/system/runtime.slice")
}

func configureDockerServiceCgroup() (bool, error) {
	if err := os.MkdirAll("/etc/systemd/system/docker.service.d", 0755); err != nil {
		return false, err
	}
	return replaceIfChanged("/configs/docker-10-cgroup.conf", "/etc/systemd/system/docker.service.d/10-cgroup.conf")
}

func configureKubeletServiceCgroup() (bool, error) {
	if err := os.MkdirAll("/etc/systemd/system/kubelet.service.d", 0755); err != nil {
		return false, err
	}
	return replaceIfChanged("/configs/kubelet-10-cgroup.conf", "/etc/systemd/system/kubelet.service.d/10-cgroup.conf")
}

func configureContainerdServiceCgroup() (bool, error) {
	if err := os.MkdirAll("/etc/systemd/system/containerd.service.d", 0755); err != nil {
		return false, err
	}
	return replaceIfChanged("/configs/containerd-10-cgroup.conf", "/etc/systemd/system/containerd.service.d/10-cgroup.conf")
}

func configureCGroups() error {
	_, err := configureRuntimeSlice()
	if err != nil {
		return err
	}
	kChanged, err := configureKubeletServiceCgroup()
	if err != nil {
		return err
	}
	dChanged, err := configureDockerServiceCgroup()
	if err != nil {
		return err
	}
	_, err = configureContainerdServiceCgroup()
	if err != nil {
		return err
	}
	if kChanged || dChanged {
		if err := daemonReload(); err != nil {
			return err
		}
		if err := restartDocker(); err != nil {
			return err
		}
		if err := restartKubelet(); err != nil {
			return err
		}
	}
	return nil
}

func configureContainerd() error {
	if err := os.MkdirAll("/etc/containerd", 0755); err != nil {
		return err
	}
	if err := os.MkdirAll("/etc/systemd/system/containerd.service.d", 0755); err != nil {
		return err
	}
	if err := os.MkdirAll("/etc/cni/net.d", 0755); err != nil {
		return err
	}
	configChanged, err := replaceIfChanged("/configs/config.toml", "/etc/containerd/config.toml")
	if err != nil {
		return err
	}
	serviceChanged, err := replaceIfChanged("/configs/containerd.service", "/etc/systemd/system/containerd.service")
	if err != nil {
		return err
	}
	cniChanged, err := replaceIfChanged("/configs/kubenet.conf", "/etc/containerd/kubenet.conf")
	if err != nil {
		return err
	}
	if configChanged || serviceChanged || cniChanged {
		if err := enableContainerd(); err != nil {
			return err
		}
		if err := daemonReload(); err != nil {
			return err
		}
		if err := restartContainerd(); err != nil {
			return err
		}
		if err := updateContainerd(); err != nil {
			return err
		}
	}
	return nil
}

func configureKubelet() error {
	if err := os.MkdirAll("/etc/systemd/system/kubelet.service.d", 0755); err != nil {
		return err
	}
	serviceChanged, err := replaceIfChanged("/configs/kubelet.service", "/etc/systemd/system/kubelet.service")
	if err != nil {
		return err
	}
	confChanged, err := replaceIfChanged("/configs/10-kubeadm.conf", "/etc/systemd/system/kubelet.service.d/10-kubeadm.conf")
	if err != nil {
		return err
	}
	flagsChanged, err := replaceIfChanged("/configs/flags.env", "/var/lib/kubelet/flags.env")
	if err != nil {
		return err
	}
	configChanged, err := replaceIfChanged("/configs/config.yaml", "/var/lib/kubelet/config.yaml")
	if err != nil {
		return err
	}
	if serviceChanged || confChanged || flagsChanged || configChanged {
		if err := daemonReload(); err != nil {
			return err
		}
		if err := restartKubelet(); err != nil {
			return err
		}
		if err := reboot(); err != nil {
			return err
		}
	}
	return nil
}

func runInit() error {
	if err := configureDockerDaemon(); err != nil {
		return err
	}
	if err := configureCGroups(); err != nil {
		return err
	}
	if err := configureContainerd(); err != nil {
		return err
	}
	if err := configureKubelet(); err != nil {
		return err
	}
	return nil
}

func main() {
	klog.InitFlags(nil)
	flag.Set("v", "3") //nolint: errcheck
	flag.Parse()
	args := os.Args[1:]
	exitCode := 0
	defer func() {
		klog.Flush()
		os.Exit(exitCode)
	}()

	if len(args) > 0 && args[0] == "init" {
		if err := runInit(); err != nil {
			klog.Error(err)
			exitCode = 1
			return
		}
		return
	}

	klog.Info("Host successfully configured")
	for {
		<-time.After(time.Duration(math.MaxInt64))
	}
}
