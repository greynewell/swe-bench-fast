# swe-bench-fast

A Go reimplementation of the [SWE-bench](https://www.swebench.com/SWE-bench/) evaluation harness with native ARM64 container support. Single static binary. No Python environment required.

## 6.3x test runner speedup on ARM64

SWE-bench's pre-built Docker images are x86_64-only. On ARM64 hosts (Apple Silicon, AWS Graviton), every container runs through QEMU emulation. Native ARM64 images eliminate that overhead entirely.

| Instance | ARM64 native (s) | x86 emulated (s) | Speedup |
|---|---|---|---|
| django__django-13346 | 2.7 | 18.9 | 7.0x |
| matplotlib__matplotlib-14623 | 38.0 | 265.7 | 7.0x |
| scikit-learn__scikit-learn-25102 | 2.7 | 18.2 | 6.6x |
| pytest-dev__pytest-6197 | 4.7 | 28.2 | 6.1x |
| sympy__sympy-11618 | 1.9 | 8.0 | 4.2x |
| **Total (11 instances)** | **87.3** | **551.7** | **6.3x** |

Full results and methodology: [benchmark gist](https://gist.github.com/greynewell/497005bb33641503f1a5874f16578088)

Blog post with detailed writeup: [SWE-bench Tests Run 6x Faster on ARM64 with Native Containers](https://greynewell.com/blog/swe-bench-arm64-native-containers-6x-faster/)

## 78% of SWE-bench runs natively on ARM64

Out of 2,294 instances in the full dataset, 1,798 build and run natively on ARM64. Only 496 require x86 (mostly scikit-learn, matplotlib, and xarray) due to binary conda packages. Those still work via QEMU.

## Quick start

```bash
# Build the binary
make build

# Build native ARM64 images for a dataset
./dist/swe-bench-fast build --dataset swe-bench-arm64.jsonl --workers 4

# Run evaluations with pre-computed predictions
./dist/swe-bench-fast run --dataset swe-bench-arm64.jsonl --predictions preds.jsonl

# Build and push to a registry
./dist/swe-bench-fast build --dataset swe-bench-arm64.jsonl --push docker.io/youruser/swe-bench-arm64
```

### On Apple Silicon

```bash
# Docker Desktop or Colima with at least 120 GB disk, 8+ CPU cores
# Architecture is auto-detected (aarch64)
./dist/swe-bench-fast build --dataset swe-bench-arm64.jsonl
```

### On AWS Graviton

```bash
# c7g, m7g, r7g, or Graviton4 r8g instances
# Already linux/arm64, no VM layer needed
# Install QEMU for the 22% of instances that require x86:
sudo apt install qemu-user-static
./dist/swe-bench-fast build --dataset swe-bench-full.jsonl
```

## How it works

Three-layer Docker image pipeline: **base** (1-2 unique) -> **env** (~60 unique) -> **instance** (2,294 total). Each layer is deduplicated and built in parallel with BuildKit. Images are tagged deterministically so builds are idempotent.

The eval runner applies a patch, executes the test suite, parses pytest output, and grades results against fail-to-pass / pass-to-pass expectations. Same semantics as the upstream Python harness.

## Pre-built images

Pre-built ARM64 images are available on [Docker Hub](https://hub.docker.com/repository/docker/greynewell/swe-bench-fast/general). The x86-only instances (496 of 2,294) are not included; use the Epoch x86 images for those.

## CI

The repo includes a GitHub Actions workflow (`build-images.yml`) that builds and pushes ARM64 images via matrixed jobs on ARM64 runners. It splits the dataset into configurable chunks, filters out images that already exist on Docker Hub, and pushes each image individually so partial runs are resumable.

## Built with MIST

swe-bench-fast is built on the [MIST stack](https://miststack.dev) (Modular Integrated Software Toolkit), a Go framework for CLI applications with structured configuration, lifecycle management, and parallel execution primitives.
