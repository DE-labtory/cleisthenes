package util

import (
	"net"
	"strconv"
)

func GetAvailablePort(startPort uint16) uint16 {
	portNumber := startPort
	for {
		strPortNumber := strconv.Itoa(int(portNumber))
		lis, err := net.Listen("tcp", "127.0.0.1:"+strPortNumber)
		if err == nil {
			lis.Close()
			return portNumber
		}
		portNumber++
	}
}
