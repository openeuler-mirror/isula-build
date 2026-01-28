# isula-build

isula-build is a tool provided by the iSula team for building container images. It can quickly build a container image based on a given `Dockerfile`.

The tool adopts the server + client mode. The binary file `isula-build` is the client that provides a CLI for building and managing images, while `isula-builder` is the server that runs as a daemon in the background, responding all the requests from client.

User can use the CLI to

- Build an image from a Dockerfile (build).
- List all images in local store (image).
- Import container base images (import).
- Load layered images (load).
- Remove local persistent images (rm).
- Export layered images (save).
- Tag local persistent images (tag).
- Pull images from a remote repository (pull).
- Push images to a remote repository (push).
- View operating environment and system information (info).
- Log in to a remote image repository (login).
- Log out of a remote image repository (logout).
- Query isula-build version (version).

In addition, the following capabilities are also provided:

- Dockerfile compatible syntax.
- Support for extended file attributes, such as linux security, IMA, EVM, user, and trusted.
- Support for export of different image formats, for example, docker-archive, iSulad.

## Documentation

- 📖 [Container Image Building](./doc/manual_en.md) - Complete user manual
- 📚 [Usage Guidelines](./doc/manual_en.md#usage-guidelines) - Detailed usage instructions
- 🔧 [Integration Guide](./doc/manual_en.md#directly-integrating-a-container-engine) - Integration with container engines

## Getting Started

### Quick Links

- 📖 [User Manual](./doc/manual_en.md) - Complete documentation
- 🛠️ [Build Guide](#installation-on-openeuler) - Installation instructions
- 🔗 [Integration](./doc/manual_en.md#directly-integrating-a-container-engine) - Container engine integration
- ⚠️ [Precautions](./doc/manual_en.md#precautions) - Important notes and limitations

### Development Environment Setup

#### Prerequisites

- **OS**: Linux (recommended: openEuler, CentOS, Ubuntu)
- **Go**: Version 1.15 or later
- **Build Tools**: make, git
- **Dependencies**: See installation section below

#### Quick Setup

1. **Clone the repository:**
   ```bash
   git clone https://gitee.com/openeuler/isula-build.git
   cd isula-build
   ```

2. **Install dependencies (openEuler):**
   ```bash
   sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel \
       gpgme-devel libassuan-devel libseccomp-devel git bzip2 \
       go-md2man systemd-devel golang
   ```

3. **Build from source:**
   ```bash
   sudo make all
   sudo make install
   ```

For other distributions, see the installation section below.

#### IDE Configuration

**Recommended IDEs:**
- **GoLand/IntelliJ IDEA**: Open project, Go plugin will auto-configure
- **VSCode**: Install Go extension, configure with `go.toolsEnvVars`
- **Vim/Neovim**: Use gopls (Go language server) with vim-go or coc.nvim

**Go Development Setup:**
```bash
# Install Go tools
go install golang.org/x/tools/gopls@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Verify Go environment
go env
```

**Code Quality Tools:**
- Static analysis: `hack/static_check.sh all` (full check) or `hack/static_check.sh last` (latest commit)
- Uses `golangci-lint` with configuration in `.golangci.yml`
- Run tests: `make test` or `go test ./...`

### Installation on openEuler

#### Install from source.

For compiling from source on openEuler, these packages are required on user's OS:

- make
- golang (version 1.15 or later)
- btrfs-progs-devel
- device-mapper-devel
- glib2-devel
- gpgme-devel
- libassuan-devel
- libseccomp-devel
- git
- bzip2
- systemd-devel

User can install them on openEuler with `yum`:

```sh
sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel gpgme-devel libassuan-devel libseccomp-devel git bzip2 go-md2man systemd-devel golang
```

Get the source code with `git`:

```sh
git clone https://gitee.com/openeuler/isula-build.git
```

Enter the source code directory and begin compiling:

```sh
cd isula-build
sudo make
```
Five targets can be selected to compile under "Build" after the command is executed:
- all
- isula-build
- isula-builder
- safe
- debug

`sudo make all` compiles the client isula-build and server isula-builder. `sudo make isula-build` compiles only the client. `sudo make isula-builder` compiles only the server. `sudo make safe` compiles the client and server to generate secure binary files. `sudo make debug` compiles the client and server to  generate binary files containing debug. User can select targets to compile as required.

After compiling success, user can install the binaries and default configuration files simply with:

```sh
sudo make install
```

#### Install as RPM package.

`isula-build` is now released with update pack of openEuler 20.03 LTS, user can install it using yum or rpm. Before user install, please enable "update" in the repo file.

##### With `yum`

```sh
sudo yum install -y isula-build
```

**NOTE**: Please make sure the "update" part of user's yum configuration is enabled. User can download the source of yum from [openEuler repo list](https://repo.openeuler.org/) and install it.

##### With `rpm`

User can download the RPM package of isula-build and intall it.

```sh
sudo rpm -ivh isula-build-*.rpm
```

### Running the Daemon Server

#### Run as the system service (Recommended)

To manage `isula-build` by systemd, follow these steps:

```bash
# Install systemd service file
sudo install -p -m 640 ./isula-build.service /etc/systemd/system/isula-build.service

# Enable and start the service
sudo systemctl enable isula-build
sudo systemctl start isula-build

# Check service status
sudo systemctl status isula-build
```

#### Directly run the isula-builder binary

Alternatively, you can run the isula-builder binary directly:

```bash
sudo isula-builder --dataroot="/var/lib/isula-build"
```

**Note**: When running directly, ensure the daemon is stopped if it's running as a systemd service:
```bash
sudo systemctl stop isula-build
```

### Example on Building Container Images

#### Prerequisites

For building container images, `runc` is required.

User can get `runc` by installing `docker` or `docker-runc` on user's openEuler distro:

```sh
sudo yum install docker
```

or

```sh
sudo yum install docker-runc
```

#### Build an image.

Here is an example for building a container image, for more details please refer to [Usage Guidelines](./doc/manual_en.md#usage-guidelines).

Create a simple buildDir and write the Dockerfile

```dockerfile
FROM alpine:latest
LABEL foo=bar
COPY ./* /home/dir1/
```

Build the image in the buildDir.

```sh
$ sudo isula-build ctr-img build -f Dockerfile .
STEP  1: FROM alpine:latest
STEP  2: LABEL foo=bar
STEP  3: COPY ./* /home/dir1/
Getting image source signatures
Copying blob sha256:e9235582825a2691b1c91a96580e358c99acfd48082cbf1b92fd2ba4a791efc3
Copying blob sha256:dc3bca97af8b81508c343b13a08493c7809b474dc25986fcbae90c6722201be3
Copying config sha256:9ec92a8819f9da1b06ea9ff83307ff859af2959b70bfab101f6a325b1a211549
Writing manifest to image destination
Storing signatures
Build success with image id: 9ec92a8819f9da1b06ea9ff83307ff859af2959b70bfab101f6a325b1a211549
```

#### List local images.

```sh
$ sudo isula-build ctr-img images
-----------------  -----------  ----------------  ----------------------------------------------
    REPOSITORY         TAG          IMAGE ID                       CREATED
------------------  ----------  ----------------  ----------------------------------------------
      <none>          latest      9ec92a8819f9        2020-06-11 07:45:39.265106109 +0000 UTC
```

### Removing Images

```sh
$ sudo isula-build ctr-img rm 9ec92a8819f9
Deleted: sha256:86567f7a01b04c662a9657aac436e8d63ecebb26da4252abb016d177721fa11b
```

### Integration with iSulad or Docker

Integration with `iSulad` or `docker` are listed in [integration](./doc/manual_en.md#directly-integrating-a-container-engine).

## Precautions

Constraints, limitations, and differences from `docker build` are listed in [precautions](./doc/manual_en.md#precautions).

## Troubleshooting

### Common Issues

#### Build Issues

**Problem**: `go: command not found`
```bash
# Solution: Install Go (version 1.15+)
# On openEuler:
sudo yum install golang

# Or download from https://golang.org/dl/
```

**Problem**: Missing dependencies during build
```bash
# Solution: Install all required development packages
sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel \
    gpgme-devel libassuan-devel libseccomp-devel git bzip2 \
    go-md2man systemd-devel golang
```

**Problem**: Build fails with permission errors
```bash
# Solution: Ensure you have sudo privileges for installation
# Some build steps require root access
sudo make install
```

#### Runtime Issues

**Problem**: Cannot connect to isula-builder daemon
```bash
# Solution: Check if daemon is running
sudo systemctl status isula-build

# Or check if process is running
ps aux | grep isula-builder

# Start the daemon if not running
sudo systemctl start isula-build
```

**Problem**: Permission denied when building images
```bash
# Solution: Ensure you have proper permissions
# isula-build typically requires sudo for building
sudo isula-build ctr-img build -f Dockerfile .
```

**Problem**: `runc` not found during build
```bash
# Solution: Install runc or docker
sudo yum install docker
# or
sudo yum install docker-runc
```

#### Development Issues

**Problem**: Static check fails
```bash
# Solution: Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run static check
hack/static_check.sh all
```

**Problem**: Tests fail
```bash
# Solution: Ensure all dependencies are installed
# Run tests with verbose output
go test -v ./...

# Or use make
make test
```

For more detailed troubleshooting, please refer to:
- [User Manual](./doc/manual_en.md)
- [Precautions](./doc/manual_en.md#precautions)

## How to Contribute

We welcome contributions! Here's how to get started:

### Before Contributing

1. **Sign the CLA**: Please sign the [CLA](https://openeuler.org/en/cla.html) before contributing
2. **Read the documentation**: Familiarize yourself with the [user manual](./doc/manual_en.md)
3. **Check existing issues**: Look for existing issues or create a new one

### Development Workflow

1. **Fork the repository** and clone your fork
2. **Create a feature branch**: `git checkout -b feature/your-feature-name`
3. **Make your changes** and ensure they pass:
   - Static checks: `hack/static_check.sh all`
   - Tests: `make test` or `go test ./...`
4. **Commit your changes**: Use clear, descriptive commit messages
5. **Push to your fork** and create a Pull Request

### Code Quality Standards

- Follow Go coding standards and conventions
- Run `hack/static_check.sh all` before submitting
- Ensure all tests pass
- Update documentation if needed
- Add tests for new features

### Getting Help

- Check the [documentation](./doc/manual_en.md)
- Review existing issues and PRs
- Ask questions in issues or discussions

## Licensing

isula-build is licensed under the **Mulan PSL v2**.

For more information, see the [LICENSE](./License/LICENSE) file.
