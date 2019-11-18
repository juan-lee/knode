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
	"flag"
	"fmt"
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

func nsEnterCommand(arg ...string) *exec.Cmd {
	args := append([]string{"-m/proc/1/ns/mnt"}, arg...)
	return exec.Command("/usr/bin/nsenter", args...)
}

func daemonReload() error {
	cmd := nsEnterCommand("/bin/systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func restartDocker() error {
	cmd := nsEnterCommand("/bin/systemctl", "restart", "docker")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func restartKubelet() error {
	cmd := nsEnterCommand("/bin/systemctl", "restart", "kubelet")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func reboot() error {
	cmd := nsEnterCommand("/bin/systemctl", "reboot")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func configureDockerDaemon() error {
	_, err := os.Stat("/configs/daemon.json")
	if os.IsNotExist(err) {
		return nil
	}

	current, err := readFile("/etc/docker/daemon.json")
	if err != nil {
		return err
	}
	desired, err := readFile("/configs/daemon.json")
	if err != nil {
		return err
	}

	if bytes.Equal(current, desired) {
		klog.Info("/etc/docker/daemon.json already configured")
		return nil
	}

	klog.Infof("Updating /etc/docker/daemon.json:\n%s", desired)
	if err := ioutil.WriteFile("/etc/docker/daemon.json", desired, 0644); err != nil {
		return err
	}

	if err := restartDocker(); err != nil {
		return err
	}
	if err := restartKubelet(); err != nil {
		return err
	}
	return nil
}

func configureRuntimeSlice() (bool, error) {
	_, err := os.Stat("/configs/runtime.slice")
	if os.IsNotExist(err) {
		return false, nil
	}

	current, err := readFile("/etc/systemd/system/runtime.slice")
	if err != nil {
		return false, err
	}
	desired, err := readFile("/configs/runtime.slice")
	if err != nil {
		return false, err
	}

	if bytes.Equal(current, desired) {
		klog.Info("/etc/systemd/system/runtime.slice already configured")
		return false, nil
	}

	klog.Infof("Updating /etc/systemd/system/runtime.slice:\n%s", desired)
	if err := ioutil.WriteFile("/etc/systemd/system/runtime.slice", desired, 0644); err != nil {
		return false, err
	}
	return true, nil
}

func configureServiceCGroup(name string) (bool, error) {
	if _, err := os.Stat("/configs/10-cgroup.conf"); os.IsNotExist(err) {
		return false, nil
	}

	if err := os.MkdirAll(name, 0755); err != nil {
		return false, err
	}

	current, err := readFile(fmt.Sprintf("%s/10-cgroup.conf", name))
	if err != nil {
		return false, err
	}
	desired, err := readFile("/configs/10-cgroup.conf")
	if err != nil {
		return false, err
	}

	if bytes.Equal(current, desired) {
		klog.Infof("%s/10-cgroup.conf already configured", name)
		return false, nil
	}

	klog.Infof("Updating %s/10-cgroup.conf:\n%s", name, desired)
	if err := ioutil.WriteFile(fmt.Sprintf("%s/10-cgroup.conf", name), desired, 0644); err != nil {
		return false, err
	}
	return true, nil
}

func configureCGroups() error {
	rsChanged, err := configureRuntimeSlice()
	if err != nil {
		return err
	}
	kChanged, err := configureServiceCGroup("/etc/systemd/system/kubelet.service.d")
	if err != nil {
		return err
	}
	dChanged, err := configureServiceCGroup("/etc/systemd/system/docker.service.d")
	if err != nil {
		return err
	}
	if rsChanged || kChanged || dChanged {
		if err := daemonReload(); err != nil {
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
