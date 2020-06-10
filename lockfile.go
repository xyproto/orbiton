package main

import (
	"net"
	"strconv"
)

// string2int tries its best to convert a string to a number from 1024 up to 65535
func string2int(s string) int {
	port := 1024
	multiplier := 1.0
	for _, r := range s {
		port += int(float64(r) * multiplier)
		multiplier += 0.1
		if port >= 65535 {
			port = 1024
		}
	}
	return port
}

// ProbablyAlreadyOpen checks if it is likely that the given absolute filename is already open with o
func ProbablyAlreadyOpen(absFilename string) bool {
	fileLockPort := string2int(absFilename)
	// Thanks https://rosettacode.org/wiki/Determine_if_only_one_instance_is_running#Port
	_, err := net.Listen("tcp", ":"+strconv.Itoa(fileLockPort))
	// If there was an error, the port is already taken
	alreadyOpen := err != nil
	return alreadyOpen // Yes, other instances are probably running for this filename, the port could not be opened
}
