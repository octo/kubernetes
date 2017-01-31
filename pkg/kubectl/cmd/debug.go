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
	coreclient "k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset/typed/core/internalversion"
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
	sOpt := StreamOptions{
		In:  in,
		Out: out,
		Err: errOut,
	}
	cmd := &cobra.Command{
		Use:     "debug POD [-c CONTAINER] (--image|--command)",
		Short:   "Debug a pod by copying and modifying it",
		Long:    debugLong,
		Example: debugExample,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var extraArgs []string
			if l := cobraCmd.ArgsLenAtDash(); l != -1 {
				args, extraArgs = args[:l], args[l:]
			}
			dbg, err := newDebugCmd(f, cobraCmd, args, extraArgs, sOpt)
			cmdutil.CheckErr(err)

			cmdutil.CheckErr(dbg.Run())
		},
	}

	flags := cmd.Flags()

	// copied from run.go
	flags.Bool("attach", false, "If true, wait for the Pod to start running, and then attach to the Pod as if 'kubectl attach ...' were called.  Default false, unless '-i/--stdin' is set, in which case the default is true. With '--restart=Never' the exit code of the container process is returned.")
	flags.Bool("command", false, "If true and extra arguments are present, use them as the 'command' field in the container, rather than the 'args' field which is the default.")
	flags.String("image", "", "The image for the container to run.")

	// copied from exec.go
	flags.StringP("pod", "p", "", "Pod name")
	flags.StringP("container", "c", "", "Container name. If omitted, the first container in the pod will be chosen")
	flags.BoolP("stdin", "i", false, "Pass stdin to the container")
	flags.BoolP("tty", "t", false, "Stdin is a TTY")

	// flags.Bool("in-place", false, "When enabled, the pod to debug is modifyed rather a modifyed copy being created")

	return cmd
}

// debugCmd holds flags and context for executing the "debug" command.
type debugCmd struct {
	// Flags
	SrcPod     string
	DstPod     string
	Container  string
	Image      string
	EntryPoint []string
	Args       []string
	Attach     bool

	*cobra.Command
	cmdutil.Factory
	StreamOptions
}

// newDebugCmd initializes and returns a debugCmd.
func newDebugCmd(f cmdutil.Factory, cmd *cobra.Command, args, extraArgs []string, sOpt StreamOptions) (*debugCmd, error) {
	if len(args) < 1 {
		return nil, cmdutil.UsageError(cmd, "name of pod to debug is required")
	}

	sOpt.Stdin = cmdutil.GetFlagBool(cmd, "stdin")
	sOpt.TTY = cmdutil.GetFlagBool(cmd, "tty")

	attach := sOpt.Stdin // sic
	if cmd.Flags().Lookup("attach").Changed {
		attach = cmdutil.GetFlagBool(cmd, "attach")
	}

	return &debugCmd{
		SrcPod:     args[0],
		DstPod:     cmdutil.GetFlagString(cmd, "pod"),
		Container:  cmdutil.GetFlagString(cmd, "container"),
		Image:      cmdutil.GetFlagString(cmd, "image"),
		EntryPoint: args[1:],
		Args:       extraArgs,
		Attach:     attach,

		Command:       cmd,
		Factory:       f,
		StreamOptions: sOpt,
	}, nil
}

// Run executes the "debug" command.
func (cmd *debugCmd) Run() error {
	src, err := cmd.pod(cmd.SrcPod)
	if err != nil {
		return err
	}

	if cmd.DstPod == "" {
		cmd.DstPod = cmd.SrcPod + "-debug"
	}

	if cmd.Container == "" {
		cmd.Container = src.Spec.Containers[0].Name
	}

	spec, err := cmd.modifiedSpec(src.Spec)
	if err != nil {
		return err
	}

	pod := &api.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: src.ObjectMeta.Namespace,
			Name:      cmd.DstPod,
		},
		Spec: *spec,
	}

	if err := cmd.createPod(pod); err != nil {
		return err
	}

	return cmd.attachPod(pod)
}

func (cmd *debugCmd) podClient() (coreclient.PodInterface, error) {
	cs, err := cmd.Factory.ClientSet()
	if err != nil {
		return nil, err
	}

	// TODO(octo): allow to specify a namespace.
	ns, _, err := cmd.Factory.DefaultNamespace()
	if err != nil {
		return nil, err
	}

	return cs.Core().Pods(ns), nil
}

func (cmd *debugCmd) pod(name string) (*api.Pod, error) {
	client, err := cmd.podClient()
	if err != nil {
		return nil, err
	}

	pod, err := client.Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve spec for pod %q: %v", name, err)
	}

	return pod, nil
}

func (cmd *debugCmd) createPod(pod *api.Pod) error {
	client, err := cmd.podClient()
	if err != nil {
		return err
	}

	if _, err := client.Create(pod); err != nil {
		return fmt.Errorf("client.Create(%q) = %v", pod.ObjectMeta.Name, err)
	}

	fmt.Printf("pod %q created\n", pod.ObjectMeta.Name)
	return nil
}

func (cmd *debugCmd) modifiedSpec(spec api.PodSpec) (*api.PodSpec, error) {
	c, ok := cmd.container(&spec)
	if ok {
		c = cmd.modifiedContainer(c)
	} else {
		spec.Containers = append(spec.Containers, cmd.newContainer())
		c = &spec.Containers[len(spec.Containers)-1]
	}

	return &spec, nil
}

// newContainer returns a new container specification according to the debug
// command's flags and arguments.
func (cmd *debugCmd) newContainer() api.Container {
	return api.Container{
		Name:    cmd.Container,
		Image:   cmd.Image,
		Command: cmd.EntryPoint,
		Args:    cmd.Args,

		Stdin: cmd.Stdin,
		TTY:   cmd.TTY,
	}
}

// container returns a pointer to the named container, or to the first
// container if name is the empty string, and true. If no container by that
// name exists, (nil, false) is returned.
func (cmd *debugCmd) container(spec *api.PodSpec) (*api.Container, bool) {
	for i := range spec.Containers {
		if spec.Containers[i].Name == cmd.Container {
			return &spec.Containers[i], true
		}
	}

	return nil, false
}

func (cmd *debugCmd) modifiedContainer(c *api.Container) *api.Container {
	if cmd.Image != "" {
		c.Image = cmd.Image
	}
	if len(cmd.EntryPoint) != 0 {
		c.Command = cmd.EntryPoint
	}
	if len(cmd.Args) != 0 {
		c.Args = cmd.Args
	}
	c.Stdin = cmd.Stdin
	c.TTY = cmd.TTY

	return c
}

func (cmd *debugCmd) attachPod(pod *api.Pod) error {
	if !cmd.Attach {
		return nil
	}

	config, err := cmd.Factory.ClientConfig()
	if err != nil {
		return err
	}

	cs, err := cmd.Factory.ClientSet()
	if err != nil {
		return err
	}

	opts := &AttachOptions{
		StreamOptions: cmd.StreamOptions,

		// TODO(octo): Run uses "attach"; why?!
		CommandName: cmd.Command.Parent().CommandPath() + " attach",

		Pod: pod,

		Attach:    &DefaultRemoteAttach{},
		PodClient: cs.Core(),
		Config:    config,
	}
	opts.StreamOptions.ContainerName = cmd.Container

	// handleAttachPod is declared in run.go
	if err := handleAttachPod(cmd.Factory, opts.PodClient, pod.Namespace, pod.Name, opts, opts.StreamOptions.Quiet); err != nil {
		return err
	}

	// TODO(octo):
	return nil
}