package skel

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/suvarshachennareddy/minicni/common/types"
)

type CmdArgs struct {
	CniCmd         types.CNICommand
	CniContainerId string
	CniNetNs       string
	CniIfname      string
	StdinData      []byte
}

func GetArgs() (*CmdArgs, error) {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}

	args := &CmdArgs{
		StdinData: input,
	}

	args.CniCmd = types.CNICommand(os.Getenv("CNI_COMMAND"))
	if !types.IsSupportedCniCommand(args.CniCmd) {
		return nil, fmt.Errorf("unsupported CNI Command: %v", args.CniCmd)
	}

	args.CniContainerId = os.Getenv("CNI_CONTAINERID")
	if args.CniContainerId == "" {
		return nil, errors.New("empty CNI_CONTAINERID")
	}

	args.CniNetNs = os.Getenv("CNI_NETNS")
	if args.CniNetNs == "" {
		return nil, errors.New("empty CNI_NETNS")
	}

	args.CniIfname = os.Getenv("CNI_IFNAME")
	if args.CniIfname == "" {
		return nil, errors.New("empty CNI_IFNAME")
	}

	return args, nil
}
