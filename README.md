# isula-build

isula-build is a tool provided by iSula team for building container images. It can quickly build the container image according to the given `Dockerfile`.

The binary file `isula-build` is a CLI tool and `isula-builder` runs as a daemon responding all the requests from client.

It provides a command line tool that can be used to

- build an image from a Dockerfile(build)
- list all images in local store(image)
- import a basic container image(import)
- load image layers(load)
- remove specified images(rm)
- exporting images layers(save)
- tag local images(tag)
- pull image from remote repository(pull)
- push image to remote repository(push)
- view operating environment and system info(info)
- login remote image repository(login)
- logout remote image repository(logout)
- query isula-build version(version)

We also

- be compatible with Dockerfile grammar
- support extended file attributes, e.g., linux security, IMA, EVM, user, trusted
- support different image formats, e.g., docker-archive, isulad

## Documentation
- [guide](./doc/manual_en.md).
- [more usage guide](./doc/manual_en.md#usage-guidelines).

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

#### Install as RPM package

`isula-build` is now released with update pack of openEuler 20.03 LTS, you can install it by the help of yum or rpm. Before you install, please enable "update" in repo file.

##### With `yum`

```sh
sudo yum install -y isula-build
```

**NOTE**: Please make sure "update" part of your yum configuration is enabled.

##### With `rpm`

you can download it from [openEuler's yum repo of update](https://repo.openeuler.org/) to your local machine, and intall it with such command:

```sh
sudo rpm -ivh isula-build-*.rpm
```

### Run the daemon server

#### Run as system service

To manage `isula-builder` by systemd, please refer to following steps:

```sh
sudo install -p -m 640 ./isula-build.service /etc/systemd/system/isula-build.service
sudo systemctl enable isula-build
sudo systemctl start isula-build
```

#### Directly running isula-builder
You can also run the isula-builder command on the server to start the service.

```sh
sudo isula-builder --dataroot="/var/lib/isula-build"
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

Here is an example for building a container image, for more details please refer to [usage](./doc/manual_en.md#usage-guidelines).

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

Integrates with `iSulad` or `docker` are listed in [integration](./doc/manual_en.md#directly-integrating-a-container-engine).

## Precautions

Constraints, limitations and the differences from `docker build` are listed in [precautions](./doc/manual_en.md#precautions).

## How to Contribute

We are happy to provide guidance for the new contributors.

Please sign the [CLA](https://openeuler.org/en/cla.html) before contributing.

## Licensing

isula-build is licensed under the **Mulan PSL v2**.
