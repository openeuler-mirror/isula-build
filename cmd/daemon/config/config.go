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
// Create: 2020-01-20
// Description: This file is used for isula-build daemon config setting

// Package config package implements isula-build daemon config
package config

// TomlConfig defines the configuration of isula-builder
type TomlConfig struct {
	Debug    bool   `toml:"debug"`
	Group    string `toml:"group"`
	LogLevel string `toml:"loglevel"`
	Runtime  string `toml:"runtime"`
	RunRoot  string `toml:"run_root"`
	DataRoot string `toml:"data_root"`
}
