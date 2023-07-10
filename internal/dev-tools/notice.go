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

package dev_tools

import (
	"fmt"
	settings "github.com/elastic/assetbeat/cmd"
	"github.com/elastic/elastic-agent-libs/dev-tools/mage"
	"github.com/elastic/elastic-agent-libs/dev-tools/mage/gotool"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"os"
)

func GenerateNotice(overrides, rules, noticeTemplate string) error {
	mg.Deps(mage.InstallGoNoticeGen)
	depsFile := generateDepsFile()
	defer os.Remove(depsFile)

	generator := gotool.NoticeGenerator
	return generator(
		generator.Dependencies(depsFile),
		generator.IncludeIndirect(),
		generator.Overrides(overrides),
		generator.Rules(rules),
		generator.NoticeTemplate(noticeTemplate),
		generator.NoticeOutput("NOTICE.txt"),
	)
}

func GenerateDependencyReport(overrides, rules, dependencyReportTemplate string, isSnapshot bool) error {
	mg.Deps(mage.InstallGoNoticeGen)
	depsFile := generateDepsFile()
	defer os.Remove(depsFile)

	if err := sh.RunV("mkdir", "-p", defaultPackageFolder); err != nil {
		return err
	}

	generator := gotool.NoticeGenerator
	dependencyReportFilename := fmt.Sprintf("dependencies-%s", settings.Version)
	if isSnapshot {
		dependencyReportFilename = dependencyReportFilename + "-SNAPSHOT"
	}
	return generator(
		generator.Dependencies(depsFile),
		generator.IncludeIndirect(),
		generator.Overrides(overrides),
		generator.Rules(rules),
		generator.NoticeTemplate(dependencyReportTemplate),
		generator.NoticeOutput(fmt.Sprintf("%s/%s.csv", defaultPackageFolder, dependencyReportFilename)),
	)
}

func generateDepsFile() string {

	out, _ := gotool.ListDepsForNotice()
	depsFile, _ := os.CreateTemp("", "depsout")
	_, _ = depsFile.Write([]byte(out))
	depsFile.Close()

	return depsFile.Name()
}
