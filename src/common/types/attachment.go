package types

import (
	"fmt"
	"strings"
)

type Route struct {
	Dst     string `json:"dst"`
	Gateway string `json:"gw,omitempty"`
}

type Interface struct {
	Name    string `json:"name"`
	Mac     string `json:"mac"`
	Sandbox string `json:"sandbox,omitempty"`
}

type IP struct {
	Address   string `json:"address"`
	Gateway   string `json:"gateway"`
	Interface *int   `json:"interface,omitempty"`
}

type AttachmentKey struct {
	ContainerID string
	IfName      string
}

func (k AttachmentKey) String() []byte {
	return []byte(k.ContainerID + ":" + k.IfName)
}

func (k *AttachmentKey) ParseString(attachmentKeyStr string) error {
	s := strings.SplitN(attachmentKeyStr, ":", 2)
	if len(s) != 2 {
		return fmt.Errorf("invalid attachment key string format: %v", attachmentKeyStr)
	}

	k.ContainerID = s[0]
	k.IfName = s[1]

	return nil
}

func (k AttachmentKey) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *AttachmentKey) UnmarshalText(data []byte) error {
	return k.ParseString(string(data))
}

type AttachmentBody struct {
	Ips        []IP        `json:"ips"`
	Interfaces []Interface `json:"interfaces"`
	Routes     []Route     `json:"routes"`
}
