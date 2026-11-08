//go:build linux

package main

import (
	"log"

	"github.com/suvarshachennareddy/minicni/common/types"
	"github.com/suvarshachennareddy/minicni/plugins/ipam/hostlocal/internal"
	"github.com/suvarshachennareddy/minicni/plugins/skel"
)

func main() {
	cniArgs, err := skel.GetArgs()
	if err != nil {
		log.Fatalf("could not get args: %v", err)
	}

	switch cniArgs.CniCmd {
	case types.CNIAdd:
		err = internal.HandleAdd(cniArgs)
	case types.CNIDel:
		err = internal.HandleDel(cniArgs)
	case types.CNICheck:
		err = internal.HandleCheck(cniArgs)
	default:
		log.Fatalf("unsupported CNI command: %v", cniArgs.CniCmd)
	}

	if err != nil {
		log.Fatalf("could not handle %s for (%s, %s): %v", cniArgs.CniCmd, cniArgs.CniIfname, cniArgs.CniNetNs, err)
	}
}
