// Copyright (c) Huawei Technologies Co., Ltd. 2020. All rights reserved.
// isula-build licensed under the Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//     http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR
// PURPOSE.
// See the Mulan PSL v2 for more details.
// Author: iSula Team
// Create: 2020-1-20
// Description: This file contains default constants used in the project

// Package constant stores constants used for the project
package constant

import "time"

const (
	// ConfigurationPath is isula-build configuration path
	ConfigurationPath = "/etc/isula-build/configuration.toml"
	// RegistryConfigPath describes the config path of registries
	RegistryConfigPath = "/etc/isula-build/registries.toml"
	// SignaturePolicyPath describes the policy path
	SignaturePolicyPath = "/etc/isula-build/policy.json"
	// StorageConfigPath describes the storage path
	StorageConfigPath = "/etc/isula-build/storage.toml"
	// RegistryDirPath is the dir to store registry configs
	RegistryDirPath = "/etc/isula-build/registries.d"
	// AuthFilePath is authentication file used for registry connection
	AuthFilePath = "/etc/isula-build/auth.json"
	// DefaultGRPCAddress is the local unix socket used by isula-builder
	DefaultGRPCAddress = "unix:///var/run/isula_build.sock"
	// UnixPrefix is the prefix used on defined an unix sock
	UnixPrefix = "unix://"
	// DefaultDataRoot is the default persistent data root used by isula-builder
	DefaultDataRoot = "/var/lib/isula-build"
	// DefaultRunRoot is the default run root used by isula-builder
	DefaultRunRoot = "/var/run/isula-build"
	// DefaultSharedDirMode is dir perm mode with higher permission
	DefaultSharedDirMode = 0755
	// DefaultSharedFileMode is file perm mode with higher permission
	DefaultSharedFileMode = 0644
	// DefaultRootFileMode is the default root file mode
	DefaultRootFileMode = 0600
	// DefaultRootDirMode is the default root dir mode
	DefaultRootDirMode = 0700
	// DefaultUmask is the working umask in isula-builder
	DefaultUmask = 0027
	// CliLogBufferLen is log channel buffer size
	CliLogBufferLen = 8
	// HostsFilePath is the path of file hosts
	HostsFilePath = "/etc/hosts"
	// ResolvFilePath is the path of file resolv.conf
	ResolvFilePath = "/etc/resolv.conf"
	// MaxFileSize is the maximum file size allowed, set 1M
	MaxFileSize = 1024 * 1024
	// DefaultHTTPTimeout includes the total time of dial, TLS handshake, request, resp headers and body
	DefaultHTTPTimeout = 3600 * time.Second
	// DefaultFailedCode is the exit code for most scenes
	DefaultFailedCode = 1
	// DefaultIDLen is the ID length for image ID and build ID
	DefaultIDLen = 12
	// DefaultCertRoot is path of certification used for registry connection
	DefaultCertRoot = "/etc/isula-build/certs.d"
	// LayoutTime is the time format used to parse time from a string
	LayoutTime = "2006-01-02 15:04:05"
	// BuildContainerImageType is the default build type
	BuildContainerImageType = "ctr-img"
	// BufferSize is default buffer size for file transportation
	BufferSize = 32 * 1024
)

var (
	// MaskedPaths is the list of paths which should be masked in container
	MaskedPaths = []string{
		"/proc/acpi",
		"/proc/config.gz",
		"/proc/kcore",
		"/proc/keys",
		"/proc/latency_stats",
		"/proc/timer_list",
		"/proc/timer_stats",
		"/proc/sched_debug",
		"/proc/scsi",
		"/proc/files_panic_enable",
		"/proc/fdthreshold",
		"/proc/fdstat",
		"/proc/fdenable",
		"/proc/signo",
		"/proc/sig_catch",
		"/proc/kbox",
		"/proc/oom_extend",
		"/sys/firmware",
		"/proc/cpuirqstat",
		"/proc/memstat",
		"/proc/iomem_ext",
		"/proc/livepatch",
		"/proc/net_namespace",
	}

	// ReadonlyPaths is the list of paths which shoud be read-only in container
	ReadonlyPaths = []string{
		"/proc/asound",
		"/proc/bus",
		"/proc/fs",
		"/proc/irq",
		"/proc/sys",
		"/proc/sysrq-trigger",
	}

	// ReservedArgs is the list of proxy related environment variables
	ReservedArgs = map[string]bool{
		"http_proxy":  true,
		"https_proxy": true,
		"ftp_proxy":   true,
		"no_proxy":    true,
		"HTTP_PROXY":  true,
		"HTTPS_PROXY": true,
		"FTP_PROXY":   true,
		"NO_PROXY":    true,
	}
)
