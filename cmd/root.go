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
	"github.com/elastic/beats/v7/libbeat/cmd"
	"github.com/elastic/beats/v7/libbeat/cmd/instance"
	"github.com/elastic/inputrunner/beater"
	"github.com/spf13/pflag"
)

// Name of this beat
const Name = "inputrunner"

// RootCmd to handle beats cli
var RootCmd *cmd.BeatsRootCmd

// InputrunnerSettings contains the default settings for inputrunner
func InputrunnerSettings() instance.Settings {
	runFlags := pflag.NewFlagSet(Name, pflag.ExitOnError)
	return instance.Settings{
		RunFlags:      runFlags,
		Name:          Name,
		HasDashboards: false,
	}
}

// Inputrunner builds the beat root command for executing inputrunner and it's subcommands.
func Inputrunner(inputs beater.PluginFactory, settings instance.Settings) *cmd.BeatsRootCmd {
	command := cmd.GenRootCmdWithSettings(beater.New(inputs), settings)
	return command
}
