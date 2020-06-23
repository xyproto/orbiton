package main

import (
	"net"
	"strconv"
	"time"
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

// canConnect tries to connect to the given host and port
func canConnect(addr string) bool {
	// connecting to addr
	conn, err := net.DialTimeout("tcp", addr, time.Second*1.0)
	if err != nil {
		// connection did not work out
		return false
	}
	// connection worked out
	if conn != nil {
		defer conn.Close()
		return true
	}
	// connected, but the connection is nil
	// Should not happen.
	return false
}

// ProbablyAlreadyOpen checks if it is likely that the given absolute filename is already open with o
func ProbablyAlreadyOpen(absFilename string) bool {
	fileLockPort := string2int(absFilename)

	// Checking if we can connect to the fileLockPort for this absFilename
	if canConnect(":" + strconv.Itoa(fileLockPort)) {
		// yes, return
		return true
	}

	// Thanks https://rosettacode.org/wiki/Determine_if_only_one_instance_is_running#Port
	// Serve on a port, to mark this absFilename as locked
	_, err := net.Listen("tcp", ":"+strconv.Itoa(fileLockPort))

	// If there was an error, the port is already taken
	alreadyOpen := err != nil

	return alreadyOpen // Yes, other instances are probably running for this filename, the port could not be opened
}
