package types

type CNICommand string

const (
	CNIAdd   CNICommand = "ADD"
	CNIDel   CNICommand = "DEL"
	CNICheck CNICommand = "CHECK"
)

var supportedCNICommands = map[CNICommand]bool{
	CNIAdd:   true,
	CNIDel:   true,
	CNICheck: true,
}

func IsSupportedCniCommand(cmd CNICommand) bool {
	return supportedCNICommands[cmd]
}
