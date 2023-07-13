// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package cmd

import (
	"github.com/elastic/assetbeat/version"
	"github.com/spf13/pflag"

	"github.com/elastic/assetbeat/beater"
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
)

// Name of this beat
const Name = "assetbeat"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// AssetbeatSettings contains the default settings for assetbeat
func AssetbeatSettings() instance.Settings {
	runFlags := pflag.NewFlagSet(Name, pflag.ExitOnError)
	return instance.Settings{
		RunFlags:      runFlags,
		Name:          Name,
		Version:       version.GetBuildVersion(),
		HasDashboards: false,
	}
}

// Assetbeat builds the beat root command for executing assetbeat and it's subcommands.
func Assetbeat(inputs beater.PluginFactory, settings instance.Settings) *cmd.BeatsRootCmd {
	command := cmd.GenRootCmdWithSettings(beater.New(inputs), settings)
	return command
}
