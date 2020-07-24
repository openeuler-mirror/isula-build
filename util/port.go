// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: Xiang Li
// Create: 2020-04-01
// Description: port related common functions

package util

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// PortSet process raw port and return "port/proto" if input is valid
// or return "" if not
func PortSet(rawPort string) (string, error) {
	port, proto, err := validatePortProto(rawPort)
	if err != nil {
		return "", err
	}
	return newPort(port, strings.ToLower(proto))
}

// getPortProto will return raw port and proto received
// default proto is tcp
func getPortProto(rawPort string) (string, string) {
	field := strings.Split(rawPort, "/")
	switch {
	// empty port
	case len(rawPort) == 0, len(field) == 0, len(field[0]) == 0:
		return "", ""
	// no protocol, set "tcp" as default proto
	case len(field) == 1:
		return rawPort, "tcp"
	// empty protocol, set "tcp" as default proto
	case len(field[1]) == 0:
		return field[0], "tcp"
	default:
		return field[0], field[1]
	}
}

// validatePort will do validation on input port
func validatePort(rawPort string) (string, error) {
	const minPortNum, maxPortNum = 0, 65535
	portNum, err := strconv.Atoi(rawPort)
	if err != nil || portNum < minPortNum || portNum > maxPortNum {
		return "", errors.Errorf("invalid port number: %s, supported range [%d-%d]", rawPort, minPortNum, maxPortNum)
	}
	return rawPort, nil
}

// validateProto will do validation on port's protocol
// only support: tcp/udp/sctp
func validateProto(proto string) (string, error) {
	var acceptProto = []string{"tcp", "udp", "sctp"}
	for _, p := range acceptProto {
		if proto == p {
			return proto, nil
		}
	}
	return "", errors.Errorf("invalid protocol %s", proto)
}

// validatePortProto will do validation for port input
// like: 3000/tcp or 3000-5000/udp
func validatePortProto(rawPort string) (string, string, error) {
	port, proto := getPortProto(rawPort)
	for _, p := range strings.Split(port, "-") {
		if _, err := validatePort(p); err != nil {
			return "", "", err
		}
	}
	if _, err := validateProto(strings.ToLower(proto)); err != nil {
		return "", "", err
	}
	return port, proto, nil
}

func newPort(port, proto string) (string, error) {
	portStart, portEnd, err := portRange(port)
	if err != nil {
		return "", err
	}
	if portStart == portEnd {
		return fmt.Sprintf("%d/%s", portStart, proto), nil
	}
	return fmt.Sprintf("%d-%d/%s", portStart, portEnd, proto), nil
}

func portRange(ports string) (int, int, error) {
	if ports == "" {
		return 0, 0, errors.Errorf("parse port failed, empty port found")
	}
	if !strings.Contains(ports, "-") {
		start, err := strconv.Atoi(ports)
		end := start
		return start, end, err
	}
	field := strings.Split(ports, "-")
	if len(field) != 2 {
		return 0, 0, errors.Errorf("parse port failed, invalid port range")
	}
	start, err := strconv.Atoi(field[0])
	if err != nil {
		return 0, 0, err
	}
	end, err := strconv.Atoi(field[1])
	if err != nil {
		return 0, 0, err
	}
	if end < start {
		return 0, 0, errors.Errorf("invalid range for port: %s", ports)
	}
	return start, end, nil
}
