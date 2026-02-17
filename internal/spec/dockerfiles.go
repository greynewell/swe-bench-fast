package spec

import (
	"fmt"
	"strings"
)

// GenerateBaseDockerfile creates the Dockerfile for the base layer:
// Ubuntu 22.04 + miniconda + system dependencies.
func GenerateBaseDockerfile(arch string) string {
	var b strings.Builder

	b.WriteString("FROM ubuntu:22.04\n\n")

	// System dependencies
	b.WriteString("ENV DEBIAN_FRONTEND=noninteractive\n")
	b.WriteString("RUN apt-get update && apt-get install -y --no-install-recommends \\\n")
	b.WriteString("    git \\\n")
	b.WriteString("    curl \\\n")
	b.WriteString("    wget \\\n")
	b.WriteString("    build-essential \\\n")
	b.WriteString("    ca-certificates \\\n")
	b.WriteString("    libffi-dev \\\n")
	b.WriteString("    libssl-dev \\\n")
	b.WriteString("    zlib1g-dev \\\n")
	b.WriteString("    libbz2-dev \\\n")
	b.WriteString("    libreadline-dev \\\n")
	b.WriteString("    libsqlite3-dev \\\n")
	b.WriteString("    libncurses5-dev \\\n")
	b.WriteString("    libxml2-dev \\\n")
	b.WriteString("    libxslt1-dev \\\n")
	b.WriteString("    libjpeg-dev \\\n")
	b.WriteString("    libpng-dev \\\n")
	b.WriteString("    libfreetype6-dev \\\n")
	b.WriteString("    pkg-config \\\n")
	b.WriteString("    locales \\\n")
	b.WriteString("    && rm -rf /var/lib/apt/lists/*\n\n")

	// Locale setup
	b.WriteString("RUN locale-gen en_US.UTF-8\n")
	b.WriteString("ENV LANG=en_US.UTF-8\n")
	b.WriteString("ENV LC_ALL=en_US.UTF-8\n\n")

	// Install miniforge (conda-forge by default, no Anaconda TOS)
	condaArch := "x86_64"
	if arch == "aarch64" || arch == "arm64" {
		condaArch = "aarch64"
	}
	b.WriteString(fmt.Sprintf("RUN curl -fsSL https://github.com/conda-forge/miniforge/releases/latest/download/Miniforge3-Linux-%s.sh -o /tmp/miniforge.sh && \\\n", condaArch))
	b.WriteString("    bash /tmp/miniforge.sh -b -p /opt/miniconda3 && \\\n")
	b.WriteString("    rm /tmp/miniforge.sh\n\n")

	b.WriteString("ENV PATH=/opt/miniconda3/bin:$PATH\n")
	b.WriteString("RUN conda init bash && conda config --set changeps1 false && \\\n")
	b.WriteString("    conda config --remove channels defaults 2>/dev/null; \\\n")
	b.WriteString("    conda config --add channels conda-forge && \\\n")
	b.WriteString("    conda config --set channel_priority strict\n\n")

	b.WriteString("WORKDIR /testbed\n")
	b.WriteString("CMD [\"/bin/bash\"]\n")

	return b.String()
}

// GenerateEnvDockerfile creates the Dockerfile for the env layer:
// FROM base, install conda env + pip packages.
func GenerateEnvDockerfile(baseTag string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM %s\n\n", baseTag))
	b.WriteString("COPY setup_env.sh /tmp/setup_env.sh\n")
	b.WriteString("RUN chmod +x /tmp/setup_env.sh && /tmp/setup_env.sh\n")

	return b.String()
}

// GenerateInstanceDockerfile creates the Dockerfile for the instance layer:
// FROM env, clone repo, checkout commit, run install.
func GenerateInstanceDockerfile(envTag string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("FROM %s\n\n", envTag))
	b.WriteString("COPY setup_repo.sh /tmp/setup_repo.sh\n")
	b.WriteString("RUN chmod +x /tmp/setup_repo.sh && /tmp/setup_repo.sh\n\n")
	b.WriteString("WORKDIR /testbed\n")

	return b.String()
}
