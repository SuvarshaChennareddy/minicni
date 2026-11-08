package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/suvarshachennareddy/minicni/common/cio"
	"github.com/suvarshachennareddy/minicni/common/constants"
	"github.com/suvarshachennareddy/minicni/common/proto"
	"github.com/suvarshachennareddy/minicni/common/types"
)

func handleClientConn(conn net.Conn, cniArgs *proto.CNIArgs) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.Close()

	var err error

	_, err = cio.WriteObj(conn, cniArgs)
	if err != nil {
		log.Printf("Failed to write to socket: %v", err)

		return
	}

	status := proto.StatusResponse{}
	_, err = cio.ReadObj(conn, &status)
	if err != nil {
		log.Printf("Failed to read from socket: %v", err)

		return
	}

	log.Printf("Status Recieved: %+v", status)
}

func extractParams(args []string) (*proto.CNIArgs, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("expected at least 4 args, got %v", len(args))
	}

	cniCmd := types.CNICommand(args[1])
	if !types.IsSupportedCniCommand(cniCmd) {
		return nil, fmt.Errorf("invalid CNI Command: %s", args[1])
	}

	cniArgs := &proto.CNIArgs{
		CniCommand:  cniCmd,
		NetworkName: args[2],
		NetNsPath:   args[3],
	}

	if len(args) >= 5 {
		cniArgs.InterfaceName = args[4]
	} else {
		cniArgs.InterfaceName = constants.DefaultInterfaceName
	}

	capabilityArgs := os.Getenv("CAP_ARGS")
	if capabilityArgs != "" {
		err := json.Unmarshal([]byte(capabilityArgs), &cniArgs.CapabilityArgs)
		if err != nil {
			return nil, fmt.Errorf("error marshalling CAP_ARGS: %s", capabilityArgs)
		}
	}

	return cniArgs, nil
}

func main() {
	c, err := net.Dial("unix", constants.SockPath)
	if err != nil {
		panic(err)
	}

	cniArgs, err := extractParams(os.Args)
	if err != nil {
		log.Printf("invalid input, error: %v", err)

		return
	}

	handleClientConn(c, cniArgs)
}
