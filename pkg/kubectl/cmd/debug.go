/*
Copyright 2017 The Kubernetes Authors.

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

package cmd

import (
	"fmt"
	"io"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/kubectl/cmd/templates"
	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/spf13/cobra"
)

var (
	debugLong = templates.LongDesc(`
		Debug a pod

		...
	`)

	debugExample = templates.Examples(`
		# Container does not exist, create a copy with an additional container
		kubectl debug example [-p NAME] [--container new-container] --image=debian

		# Container name exists, create a copy with a different entrypoint
		kubectl debug example [-p NAME] [--container existing-container] --command -- /bin/sh

		# Container "shell" does not exist, add a new container to running pod
		# TODO(octo): not yet supported
		kubectl debug example --in-place -c shell --image=debian`)
)

func NewCmdDebug(f cmdutil.Factory, in io.Reader, out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "debug POD [--in-place] [-c CONTAINER] (--image|--command)",
		Short:   "Debug a pod by copying and modifying it",
		Long:    debugLong,
		Example: debugExample,
		Run: func(cmd *cobra.Command, args []string) {
			l := cmd.ArgsLenAtDash()
			err := debugRun(f, in, out, errOut, cmd, args[:l], args[l:])
			cmdutil.CheckErr(err)
		},
	}

	flags := cmd.Flags()

	// copied from run.go
	flags.Bool("command", false, "If true and extra arguments are present, use them as the 'command' field in the container, rather than the 'args' field which is the default.")
	flags.String("image", "", "The image for the container to run.")

	// copied from exec.go
	flags.StringP("container", "c", "", "Container name. If omitted, the first container in the pod will be chosen")
	flags.BoolP("stdin", "i", false, "Pass stdin to the container")
	flags.BoolP("tty", "t", false, "Stdin is a TTY")

	flags.Bool("in-place", false, "When enabled, the pod to debug is modifyed rather a modifyed copy being created")

	return cmd
}

func debugRun(f cmdutil.Factory, in io.Reader, out, errOut io.Writer, cmd *cobra.Command, args, extraArgs []string) error {
	if len(args) < 1 {
		return cmdutil.UsageError(cmd, "name of pod to debug is required")
	}
	srcPodName, args := args[0], args[1:]
	fmt.Printf("srcPodName = %q\n", srcPodName)

	if len(extraArgs) != 0 {
		args = append(args, extraArgs...)
	}
	fmt.Printf("args = %+v\n", args)

	srcPod, err := loadPod(f, srcPodName)
	if err != nil {
		return err
	}

	container := containerName(cmd, &srcPod.Spec)
	fmt.Printf("container = %q\n", container)

	return nil
}

// loadPod retrieves information about a pod from the server and returns it.
func loadPod(f cmdutil.Factory, name string) (*api.Pod, error) {
	cs, err := f.ClientSet()
	if err != nil {
		return nil, err
	}

	// TODO(octo): allow to specify a namespace.
	ns, _, err := f.DefaultNamespace()
	if err != nil {
		return nil, err
	}
	podClient := cs.Core().Pods(ns)

	srcPod, err := podClient.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve spec for pod %q: %v", name, err)
	}

	fmt.Printf("srcPod.Spec = %#v\n", &srcPod.Spec)
	return srcPod, nil
}

// containerName returns the value of the "container" flag if specified or the
// name of the first container in the pod otherwise.
func containerName(cmd *cobra.Command, podSpec *api.PodSpec) string {
	if c := cmdutil.GetFlagString(cmd, "container"); c != "" {
		return c
	}

	return podSpec.Containers[0].Name
}
