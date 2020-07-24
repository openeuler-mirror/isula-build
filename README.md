# isula-build

isula-build is a tool provided by iSula team for building container images. It can quickly build the container image according to the given `Dockerfile`. 

The binary file `isula-build` is a CLI tool and `isula-builder` runs as a daemon responding all the requests from client.

It provides a command line tool that can be used to

- build an image from a Dockerfile
- list all images in local store
- remove specified images

We also

- be compatible with Dockerfile grammar
- support extended file attributes, e.g., linux security, IMA, EVM, user, trusted
- support different image formats, e.g., docker-archive, isulad


## Getting Started

### Install on openEuler

#### Install from source

For compiling from source on openEuler, these packages are required on your OS: 

- make
- golang (version 1.13 or higher)
- btrfs-progs-devel
- device-mapper-devel
- glib2-devel
- gpgme-devel
- libassuan-devel
- libseccomp-devel
- git
- bzip2
- go-md2man
- systemd-devel

You can install them on openEuler with `yum`:

```sh
sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel gpgme-devel libassuan-devel libseccomp-devel git bzip2 go-md2man systemd-devel golang
```

Get the source code with `git`:

```sh
git clone https://gitee.com/openeuler/isula-build.git
```

Please note that `isula-build` uses Go Modules to manage vendoring packages. Before compiling please make sure you can connect to the default goproxy server.

If you are working behind a proxy, please refer to [Go Module Proxy](https://proxy.golang.org) by setting `GOPROXY=yourproxy`.

Enter the source code directory and begin compiling:

```sh
cd isula-build
sudo make
```

After compiling success, you can install the binaries of `isula-build` to `/usr/bin/` simply with:

```sh
sudo make install
```

To run the server of `isula-build` for the first time, the default configuration files should be installed:  

```sh
sudo mkdir -p /etc/isula-build/ && \
install -p -m 600 ./cmd/daemon/config/configuration.toml /etc/isula-build/configuration.toml && \
install -p -m 600 ./cmd/daemon/config/storage.toml /etc/isula-build/storage.toml && \
install -p -m 600 ./cmd/daemon/config/registries.toml /etc/isula-build/registries.toml && \
install -p -m 600 ./cmd/daemon/config/policy.json /etc/isula-build/policy.json
```

#### Install as RPM package

`isula-build` is integrated with `openeuler/isula-kits`, for details on how to compile and install `isula-build` as RPM package, please refer to `isula-kits`.


### Run the daemon server

#### Run as system service

To manage `isula-builder` by systemd, please refer to following steps:

```sh
sudo install -p -m 640 ./isula-build.service /etc/systemd/system/isula-build.service
sudo systemctl enable isula-build
sudo systemctl start isula-build
```


### Example on building container images

#### Requirements

For building container images, `runc` is required. 

You can get `runc` by the help of installing `docker` or `docker-runc` on your openEuler distro by:

```sh
sudo yum install docker
```

or 

```sh
sudo yum install docker-runc
```

#### Building image

Here is an example for building a container image, for more details please refer to [usage](./doc/usage.md).

Create a simple buildDir and write the Dockerfile

```dockerfile
FROM alpine:latest
LABEL foo=bar
COPY ./* /home/dir1/
```

Build the image in the buildDir

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

#### Listing images

```sh
$ sudo isula-build ctr-img images
-----------------  -----------  ----------------  ----------------------------------------------
    REPOSITORY         TAG          IMAGE ID                       CREATED
------------------  ----------  ----------------  ----------------------------------------------
      <none>          latest      9ec92a8819f9        2020-06-11 07:45:39.265106109 +0000 UTC
```

#### Removing image

```sh
$ sudo isula-build ctr-img rm 9ec92a8819f9
Deleted: sha256:86567f7a01b04c662a9657aac436e8d63ecebb26da4252abb016d177721fa11b
```

### Integrates with iSulad or docker

Integrates with `iSulad` or `docker` are listed in [integration](./doc/integration.md).

## Precautions

Constraints, limitations and the differences from `docker build` are listed in [precautions](./doc/precautions.md).

## How to Contribute

We are happy to provide guidance for the new contributors.

Please sign the [CLA](https://openeuler.org/en/cla.html) before contributing.

## Licensing

isula-build is licensed under the Mulan PSL v2.
