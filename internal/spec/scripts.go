package spec

import (
	"fmt"
	"strings"
)

// GenerateSetupEnvScript generates the script that creates the conda environment
// and installs dependencies for a given spec.
func GenerateSetupEnvScript(s ImageSpec) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euxo pipefail\n\n")

	// Create conda environment with the specified Python version (use conda-forge only to avoid TOS)
	b.WriteString(fmt.Sprintf("conda create -n testbed python=%s -y --override-channels -c conda-forge\n", s.RepoSpec.Python))
	b.WriteString("source /opt/miniconda3/bin/activate\n")
	b.WriteString("conda activate testbed\n\n")

	// Install conda packages (if packages field is not a file reference)
	pkg := s.RepoSpec.Packages
	if pkg != "" && pkg != "requirements.txt" && pkg != "environment.yml" {
		b.WriteString(fmt.Sprintf("conda install -y %s\n\n", pkg))
	}

	// Install pip packages (quoted to prevent shell interpretation of <, >, etc.)
	if len(s.RepoSpec.PipPackages) > 0 {
		b.WriteString("python -m pip install --no-cache-dir \\\n")
		for i, pkg := range s.RepoSpec.PipPackages {
			if i < len(s.RepoSpec.PipPackages)-1 {
				b.WriteString(fmt.Sprintf("    '%s' \\\n", pkg))
			} else {
				b.WriteString(fmt.Sprintf("    '%s'\n", pkg))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// GenerateSetupRepoScript generates the script that clones the repo,
// checks out the base commit, and runs the install command.
func GenerateSetupRepoScript(s ImageSpec) string {
	var b strings.Builder
	b.WriteString("#!/bin/bash\nset -euxo pipefail\n\n")
	b.WriteString("source /opt/miniconda3/bin/activate\n")
	b.WriteString("conda activate testbed\n\n")

	// Clone and checkout (fetch specific commit if not in default branch)
	repoURL := fmt.Sprintf("https://github.com/%s.git", s.Repo)
	b.WriteString(fmt.Sprintf("git clone %s /testbed\n", repoURL))
	b.WriteString("cd /testbed\n")
	b.WriteString(fmt.Sprintf("git checkout %s 2>/dev/null || (git fetch origin %s && git checkout %s)\n",
		s.BaseCommit, s.BaseCommit, s.BaseCommit))

	// Install packages from repo files (after checkout, before pre_install)
	if s.RepoSpec.Packages == "requirements.txt" {
		if paths, ok := repoReqsPaths[s.Repo]; ok {
			for _, p := range paths {
				b.WriteString(fmt.Sprintf("if [ -f %s ]; then python -m pip install -r %s; fi\n", p, p))
			}
		}
	} else if s.RepoSpec.Packages == "environment.yml" {
		if paths, ok := repoEnvYMLPaths[s.Repo]; ok {
			for _, p := range paths {
				b.WriteString(fmt.Sprintf("if [ -f %s ]; then conda env update --file %s; fi\n", p, p))
			}
		}
	}

	// Run pre-install commands (after checkout, before install)
	for _, cmd := range s.RepoSpec.PreInstall {
		b.WriteString(cmd + "\n")
	}

	// Generate constraints file from pinned packages so pip install -e .
	// doesn't upgrade them (e.g., numpy==1.25.2 → numpy 2.0)
	pinnedPkgs := pinnedPackages(s.RepoSpec.PipPackages)
	if len(pinnedPkgs) > 0 {
		b.WriteString("cat > /tmp/constraints.txt << 'CONSTRAINTS_EOF'\n")
		for _, pkg := range pinnedPkgs {
			b.WriteString(pkg + "\n")
		}
		b.WriteString("CONSTRAINTS_EOF\n")
	}

	// Run install command
	if s.RepoSpec.Install != "" {
		install := s.RepoSpec.Install
		if len(pinnedPkgs) > 0 {
			install = addConstraintsFlag(install)
		}
		b.WriteString(install + "\n")
	}

	// Re-install PipPackages after project install to enforce our pins.
	// pip install -e . may upgrade packages (e.g., pytest 7→8, numpy 1→2).
	if len(s.RepoSpec.PipPackages) > 0 {
		b.WriteString("\n# Re-pin dependencies after project install\n")
		b.WriteString("python -m pip install --no-cache-dir \\\n")
		for i, pkg := range s.RepoSpec.PipPackages {
			if i < len(s.RepoSpec.PipPackages)-1 {
				b.WriteString(fmt.Sprintf("    '%s' \\\n", pkg))
			} else {
				b.WriteString(fmt.Sprintf("    '%s'\n", pkg))
			}
		}
	}

	return b.String()
}

// pinnedPackages returns PipPackages entries that contain version constraints.
func pinnedPackages(pkgs []string) []string {
	var pinned []string
	for _, pkg := range pkgs {
		if strings.Contains(pkg, "==") || strings.Contains(pkg, "<=") ||
			strings.Contains(pkg, "<") || strings.Contains(pkg, ">=") {
			pinned = append(pinned, pkg)
		}
	}
	return pinned
}

// addConstraintsFlag appends -c /tmp/constraints.txt to a pip install command.
func addConstraintsFlag(install string) string {
	if strings.Contains(install, "pip install") {
		return install + " -c /tmp/constraints.txt"
	}
	return install
}
