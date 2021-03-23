// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mhs/src/hansoft"
	"mhs/src/monorail"
	"mhs/src/projectsync"
	"os"
	"strings"
)

type hansoftAuth struct {
	User     string
	Password string
}

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ha, err := loadHansoftAuth("hansoft-auth.json")
	if err != nil {
		return err
	}

	m, err := monorail.New("monorail-auth.json")
	if err != nil {
		return err
	}

	monorailTint, err := m.Project("tint")
	if err != nil {
		return err
	}

	h, err := hansoft.New()
	if err != nil {
		return err
	}
	// TODO(bclayton) - Attempting to destroy the hansoft instance crashes in the .so. Investigate.
	// defer h.Destroy()

	hansoftSession, err := h.Connect("localhost", 50256, "Tint", ha.User, ha.Password)
	if err != nil {
		return err
	}
	// TODO(bclayton) - Attempting to destroy the hansoft session crashes in the .so. Investigate.
	// defer hansoftSession.Destroy()

	hansoftTint, err := hansoftTintProject(hansoftSession)
	if err != nil {
		return err
	}

	if err := projectsync.Sync(monorailTint, hansoftTint); err != nil {
		return err
	}

	return nil
}

func hansoftTintProject(s hansoft.Session) (hansoft.Project, error) {
	projects, err := s.Projects()
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		if strings.ToLower(p.Name()) == "tint" {
			return p, nil
		}
	}
	return nil, fmt.Errorf("Couldn't find the tint hansoft project")
}

func loadHansoftAuth(path string) (hansoftAuth, error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return hansoftAuth{}, fmt.Errorf("Failed to load '%v': %w", path, err)
	}
	ha := hansoftAuth{}
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&ha)
	if err != nil {
		return hansoftAuth{}, fmt.Errorf("Failed to parse '%v': %w", path, err)
	}
	return ha, nil
}
