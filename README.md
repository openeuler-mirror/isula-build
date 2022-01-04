# isula-build

isula-build is a tool provided by the iSula team for building container images. It can quickly build a container image based on a given `Dockerfile`.

The tool adopts the server + client mode. The binary file `isula-build` is the client that provides a CLI for building and managing images, while `isula-builder` is the server that runs as a daemon in the background, responding all the requests from client.

You can use the CLI to

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
- [Container Image Building](./doc/manual_en.md)
- [Usage Guidelines](./doc/manual_en.md#usage-guidelines)

## Getting Started

### Installation on openEuler

#### Install from source.

For compiling from source on openEuler, these packages are required on your OS:

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

You can install them on openEuler with `yum`:

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

After compiling success, you can install the binaries and default configuration files simply with:

```sh
sudo make install
```

#### Install as RPM package.

`isula-build` is now released with update pack of openEuler 20.03 LTS, you can install it using yum or rpm. Before you install, please enable "update" in the repo file.

##### With `yum`

```sh
sudo yum install -y isula-build
```

**NOTE**: Please make sure the "update" part of your yum configuration is enabled. You can download the source of yum from [openEuler repo list](https://repo.openeuler.org/) and install it.

##### With `rpm`

You can download the RPM package of isula-build and intall it.

```sh
sudo rpm -ivh isula-build-*.rpm
```

### Running the Daemon Server

#### Run as the system service.

To manage `isula-build` by systemd, please refer to following steps:

```sh
sudo install -p -m 640 ./isula-build.service /etc/systemd/system/isula-build.service
sudo systemctl enable isula-build
sudo systemctl start isula-build
```

#### Directly run the isula-builder binary file.
You can also run the isula-builder binary file on the server to start the service.

```sh
sudo isula-builder --dataroot="/var/lib/isula-build"
```

### Example on Building Container Images

#### Prerequisites

For building container images, `runc` is required.

You can get `runc` by installing `docker` or `docker-runc` on your openEuler distro:

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

## How to Contribute

We are happy to provide guidance for the new contributors.

Please sign the [CLA](https://openeuler.org/en/cla.html) before contributing.

## Licensing

isula-build is licensed under the **Mulan PSL v2**.
