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
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"time"

	"k8s.io/klog/v2"
)

func readFile(name string) ([]byte, error) {
	file, err := os.Open(name)
	if err != nil {
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

func restartDocker() error {
	cmd := nsEnterCommand("/bin/systemctl", "restart", "docker")
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func configureDockerDaemon() error {
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
	return nil
}

func runInit() error {
	if err := configureDockerDaemon(); err != nil {
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
