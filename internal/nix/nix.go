// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// Instead of using directory, prefer using the devbox.ProfilePath() function that ensures the directory exists.
const ProfilePath = ".devbox/nix/profile/default"

var ErrPackageNotFound = errors.New("package not found")

func PkgExists(nixpkgsCommit, pkg string) bool {
	_, found := PkgInfo(nixpkgsCommit, pkg)
	return found
}

type Info struct {
	NixName string
	Name    string
	Version string
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s", i.Name, i.Version)
}

func Exec(path string, command []string, env []string) error {
	runCmd := strings.Join(command, " ")
	cmd := exec.Command("nix-shell", path, "--run", runCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	return errors.WithStack(cmd.Run())
}

func PkgInfo(nixpkgsCommit, pkg string) (*Info, bool) {
	exactPackage := fmt.Sprintf("nixpkgs/%s#%s", nixpkgsCommit, pkg)
	if nixpkgsCommit == "" {
		exactPackage = fmt.Sprintf("nixpkgs#%s", pkg)
	}

	cmd := exec.Command("nix", "search",
		"--extra-experimental-features", "nix-command flakes",
		"--json", exactPackage)
	cmd.Stderr = os.Stderr
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		// for now, assume all errors are invalid packages.
		return nil, false /* not found */
	}
	pkgInfo := parseInfo(pkg, out)
	if pkgInfo == nil {
		return nil, false /* not found */
	}
	return pkgInfo, true /* found */
}

func parseInfo(pkg string, data []byte) *Info {
	var results map[string]map[string]any
	err := json.Unmarshal(data, &results)
	if err != nil {
		panic(err)
	}
	for _, result := range results {
		pkgInfo := &Info{
			NixName: pkg,
			Name:    result["pname"].(string),
			Version: result["version"].(string),
		}

		return pkgInfo
	}
	return nil
}
