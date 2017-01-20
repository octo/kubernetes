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
		# Container "shell" does not exist, add a new container to running pod
		kubectl debug -p example -c shell --image=debian

		# Container DNE, create a copy with an additional container
		kubectl debug -p example-copy --copy-of example -c shell --image=debian

		# Container name exists, create a copy with a different entrypoint
		kubectl debug -p example-copy --copy-of example -c example --command -- /bin/sh`)
)

func NewCmdDebug(f cmdutil.Factory, in io.Reader, out, errOut io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "debug [-p POD] --copy-of POD -c CONTAINER",
		Short:   "Debug a pod by copying and modifying it",
		Long:    debugLong,
		Example: debugExample,
		Run: func(cmd *cobra.Command, args []string) {
			err := debugRun(f, in, out, errOut, cmd, args)
			cmdutil.CheckErr(err)
		},
	}

	flags := cmd.Flags()
	flags.StringP("pod", "p", "", "Name of the pod to spawn.")
	flags.StringP("container", "c", "", "Name of the container to run.")
	flags.String("copy-of", "", "Name of the pod to base the debug pod on.")
	// TODO(octo): --command, --image, -t, -i

	return cmd
}

func debugRun(f cmdutil.Factory, in io.Reader, out, errOut io.Writer, cmd *cobra.Command, args []string) error {
	dstPod := cmdutil.GetFlagString(cmd, "pod")
	srcPod := cmdutil.GetFlagString(cmd, "copy-of")
	container := cmdutil.GetFlagString(cmd, "container")

	if dstPod == "" || srcPod == "" || container == "" {
		return cmdutil.UsageError(cmd, "-p/--pod, -c/--container and --copy-of are required to run")
	}

	fmt.Printf("dstPod = %q, srcPod = %q, container = %q\n", dstPod, srcPod, container)
	return nil
}
