<!-- vim-markdown-toc GFM -->

* [Usage](#usage)
    * [Configuration](#configuration)
    * [Starting the Service](#starting-the-service)
        * [Using systemd (RPM package installation)](#using-systemd-rpm-package-installation)
        * [Using binary file](#using-binary-file)
        * [isula-builder supported flags:](#isula-builder-supported-flags)
    * [Features](#features)
        * [Container Image Building](#container-image-building)
            * [--build-arg](#--build-arg)
            * [--build-static](#--build-static)
            * [--iidfile](#--iidfile)
            * [-o, --output](#-o---output)
            * [--proxy](#--proxy)
            * [--tag](#--tag)
            * [--cap-add](#--cap-add)
        * [Viewing a Local Persistent Image](#viewing-a-local-persistent-image)
        * [Importing a Base Image from a Tarball](#importing-a-base-image-from-a-tarball)
        * [Saving a Local Persistent Image](#saving-a-local-persistent-image)
        * [Deleting a Local Persistent Image](#deleting-a-local-persistent-image)
            * [-a, --all](#-a---all)
            * [-p, --prune](#-p---prune)
        * [Authentication of the Remote Image Repository](#authentication-of-the-remote-image-repository)
            * [-a, --all](#-a---all-1)
        * [Version query](#version-query)
        * [Tag an image](#tag-an-image)
        * [Show system information](#show-system-information)
        * [Load an image](#load-an-image)

<!-- vim-markdown-toc -->

# Usage

## Configuration

isula-builder contains the following configuration files:

- `/etc/isula-build/configuration.toml`: isula-builder overall configuration file, which is used to set the isula-builder log level, persistency and runtime directories, OCI runtime.
    - debug: specifies whether to enable the debug log function. The options are true or false
    - loglevel: specifies the log level. The value can be "debug", "info", "warn", or "error"
    - data_root: set the local persistency directory
    - run_root: set runtime data directory
    - runtime: runtime type. Currently, "runc" is supported only
- `/etc/isula-build/storage.toml`: configuration file of the local persistent storage, including the configuration of the used storage driver
    - driver: specifies the storage driver type. Currently, overlay2 is supported
    - runroot: temporary storage directory
    - graphroot: readable and writable image storage directory
    - For more settings, see [containers-storage.conf.5.md](https://github.com/containers/storage/blob/master/docs/containers-storage.conf.5.md)
- `/etc/isula-build/registries.toml`: config for registries setting, including the list of available image repositories and image repository blocklist
    - registries.search: search domain of the image repository. Only the image repositories in the list can be detected
    - registries.insecure: indicates the address of the insecure image repository that can be accessed. Image repositories in this list are not authenticated(Not recommended)
    - For more settings, see [containers-registries.conf.5.md](https://github.com/containers/image/blob/master/docs/containers-registries.conf.5.md)

Before starting the isula-builder service, user should configure the service as required. If not familiar with the configuration, we can directly use the default configuration of the RPM package to start the service.

## Starting the Service

### Using systemd (RPM package installation)

Modify the configuration in the preceding configuration file.

- `systemctl start isula-build.service`: start isula-build service
- `systemctl stop isula-build.service`: stop isula-build service
- `systemctl restart isula-build`: restart isula-build service
- `journalctl -u isula-build`: look up isula-build logs

### Using binary file

Some configurations can be set through the flag of the isula-builder. For example:

- `isula-builder --dataroot "/var/lib/isula-build" --debug=false`

### isula-builder supported flags:

```bash
      --dataroot string         persistent dir (default "/var/lib/isula-build")
  -D, --debug                   print debugging information (default true)
  -h, --help                    help for isula-builder
      --log-level string        The log level to be used. Either "debug", "info", "warn" or "error". (default "info")
      --runroot string          runtime dir (default "/var/run/isula-build")
      --storage-driver string   storage-driver (default "overlay")
      --storage-opt strings     storage driver option (default [overlay.mountopt=nodev])
      --version                 version for isula-builder
```

- -d, --debug: indicates whether to enable the debug mode
- --log-level: log level. The value can be "debug", "info", "warn" or "error". The default value is info
- --dataroot: local persistent path. The default path is /var/lib/isula-build/
- --runroot: Runtime path. The default value is /var/run/isula-build/
- --storage-driver: underlaying graphdriver type
- --storage-opt: underlying graphdriver configuration

When the command line startup parameter is the same as the configuration option in the configuration file, the command line parameter is preferentially used for startup.

## Features

### Container Image Building

`isula-build ctr-img build`

The build contains the following flags:

- --build-arg: string slice, which is used during the build process
- --build-static: string slice. Build binary equivalence. If this parameter is set, all timestamp differences and other build process differences (including the container ID and host name) will be eliminated, and a container image that meets BE requirements will be built.
- -f, --filename: string, indicates the path of the Dockerfile. If this parameter is not specified, the Dockerfile in the current path is used
- --iidfile: string, output image ID to the local file
- -o, --output: string, specifies the image export mode and path
- --proxy: bool, which inherits the proxy environment variable on the host side. The default value is true
- --tag: string, add tag to the built image
- --cap-add: string slice, which is added in RUN command during the build process

#### --build-arg

Receive parameters from the command line as parameters in the Dockerfile.

Usage:

`isula-build ctr-img build --build-arg foo=bar -f Dockerfile`

```bash
$ echo "This is bar file" > bar.txt
$ cat Dockerfile_arg
FROM busybox
ARG foo
ADD ${foo}.txt .
RUN cat ${foo}.txt
$ sudo isula-build ctr-img build --build-arg foo=bar -f Dockerfile_arg
STEP  1: FROM busybox
Getting image source signatures
Copying blob sha256:8f52abd3da461b2c0c11fda7a1b53413f1a92320eb96525ddf92c0b5cde781ad
Copying config sha256:e4db68de4ff27c2adfea0c54bbb73a61a42f5b667c326de4d7d5b19ab71c6a3b
Writing manifest to image destination
Storing signatures
STEP  2: ARG foo
STEP  3: ADD ${foo}.txt .
STEP  4: RUN cat ${foo}.txt
This is bar file
Getting image source signatures
Copying blob sha256:6194458b07fcf01f1483d96cd6c34302ffff7f382bb151a6d023c4e80ba3050a
Copying blob sha256:6bb56e4a46f563b20542171b998cb4556af4745efc9516820eabee7a08b7b869
Copying config sha256:39b62a3342eed40b41a1bcd9cd455d77466550dfa0f0109af7a708c3e895f9a2
Writing manifest to image destination
Storing signatures
Build success with image id: 39b62a3342eed40b41a1bcd9cd455d77466550dfa0f0109af7a708c3e895f9a2
```

#### --build-static

The BE (Binary Equivalence) aims to implement repeated build of the same version based on the same source code, environment and ensure that the build results are the same.

The BE must meet the following requirements:

- The build environment must be consistent, including the operating system, compiler, environment variables, and configuration information
- The image storage path in the environment is the same
- The build commands are the same
- The third-party library versions are the same

For container image building, isula-build supports the same Dockerfile. If the build environments are the same, the image content and ID generated after multiple builds are the same.

If this parameter is set to BE, all timestamp differences and other build process differences (including the container ID and host name) will be eliminated, and a container image that meets BE requirements will be built. These options are supported by `--build-static` currently:
* build-time: string. A fixed timestamp with the format of `YYYY-MM-DD HH-MM-SS` used to build a static image. The timestamp affects the file attributes of creation and modification time in the diff layer. Finally, a container image that meets BE requirements is built.

Usage:

`isula-build ctr-img build -f Dockerfile --build-static='build-time=2020-05-23 10:55:33' -o docker-archive:./my-image.tar`

#### --iidfile

Export the built image ID to a file.

Usage:

`isula-build ctr-img build --iidfile testfile`

```bash
$ sudo isula-build ctr-img build -f Dockerfile_arg --iidfile testfile
$ cat testfile
76cbeed38a8e716e22b68988a76410eaf83327963c3b29ff648296d5cd15ce7b
```

#### -o, --output

Currently, -o, --output supports the following formats:

- `isulad:image:tag`: Push the successfully built image to iSulad(The isula-build and iSulad must be on the same node and image must has tag with it)

    Example: `-o isulad:busybox:latest`

- `docker-daemon:image:tag`: Push the successfully built image to Docker daemon(The isula-build and Docker must be on the same node)

    Example: `-o docker-daemon:busybox:latest`

- `docker://registry.example.com/repository:tag`: Push the built image directly to remote image repository

    Example: `-o docker://docker.io/library/busybox:latest`

- `docker-archive:<path>/<filename>:image:tag`: Save the built image as a Docker image on the local host

    Example: `-o docker-archive:/root/image.tar:busybox:latest`

In addition, the command line of the build subcommand also receives an argument(string), which indicates the context: the context of the Dockerfile build environment. The default value of this parameter is the current path(`.`) where the isula-build command is executed. This path affects the path searched by the ADD/COPY command of .dockerignore and Dockerfile.

#### --proxy

Indicates whether the container started by running the RUN command inherits the proxy environment variable "http_proxy","https_proxy","ftp_proxy","no_proxy","HTTP_PROXY","HTTPS_PROXY","FTP_PROXY","NO_PROXY". The default value is true.

#### --tag

add tag to the built image

Usage:

`isula-build ctr-img build --tag busybox:latest`

#### --cap-add

add Linux capabilities for RUN command

Usage:

`isula-build ctr-img build --cap-add CAP_SYS_ADMIN `

### Viewing a Local Persistent Image

We can run the images command to view the image stored locally.

```bash
$ sudo isula-build ctr-img images
----------------------------------------------  -----------  -----------------  --------------------------  ------------
 REPOSITORY                                      TAG          IMAGE ID           CREATED                     SIZE
----------------------------------------------  -----------  -----------------  --------------------------  ------------
 docker.io/library/alpine                        latest       a24bb4013296       2020-20-19 19:59:19         5.85 MB
 <none>                                          <none>       39b62a3342ee       2020-20-19 20:06:38         1.45 MB
----------------------------------------------  -----------  -----------------  --------------------------  ------------
```

### Importing a Base Image from a Tarball

We can run the `import` command to import a base image into the image store from a tarball.

Usage:

`isula-build ctr-img import file [REPOSITORY[:TAG]]`

```bash
$ sudo isula-build ctr-img import busybox.tar
Import success with image id: bf7b3b8ad6d842fb6e0c2dd60727ccb60a86c0e8781a35ae39de5aeef9979189
```

```bash
$ sudo isula-build ctr-img import busybox.tar busybox:isula
Import success with image id: 2d77083e646bf77e25547ea489b00ed8ec318cc37ba81c41e7ec92bca2845033
```

### Saving a Local Persistent Image

we can run the `save` command to save the image stored locally and make it a tarball.

Usage:

`isula-build ctr-img save [REPOSITORY:TAG]|imageID -o xx.tar`

Currently, these flags are supporting:

```
Flags:
  -o, --output string   Path to save the tarball
```

Examples:

```bash
$ sudo isula-build ctr-img save busybox:latest -o busybox.tar
Save success with image: busybox:latest
```

```bash
$ sudo isula-build ctr-img save 21c3e96ac411 -o busybox.tar
Save success with image: 21c3e96ac411
```

### Deleting a Local Persistent Image

We can run the `rm` command to delete the image stored locally.

Usage:

`isula-build ctr-img rm IMAGE [IMAGE...] [FLAGS]`

Please note, we do not permit removing image without tag, such as image name "foo" will not match with "foo:latest". 

Currently, these flags are supporting:

```bash
Flags:
  -a, --all     remove all images
  -h, --help    help for rm
  -p, --prune   remove all untagged images
```

#### -a, --all

Deleting all images stored locally and persistently

#### -p, --prune

Deleting all images that do not have tags and are stored locally and persistently.

```bash
$ sudo isula-build ctr-img rm -p
Deleted: sha256:78731c1dde25361f539555edaf8f0b24132085b7cab6ecb90de63d72fa00c01d
Deleted: sha256:eeba1bfe9fca569a894d525ed291bdaef389d28a88c288914c1a9db7261ad12c
```

### Tag an image

We can use the `tag` command to add an additional tag to an image.

Usage:

`isula-build ctr-img tag <imageID>/<imageName> busybox:latest`

```bash
$ sudo isula-build ctr-img images
----------------------------------------------  -----------  -----------------  --------------------------  ------------
 REPOSITORY                                      TAG          IMAGE ID           CREATED                     SIZE
----------------------------------------------  -----------  -----------------  --------------------------  ------------
 docker.io/library/alpine                        latest       a24bb4013296       2020-05-29 21:19:46         5.85 MB
----------------------------------------------  -----------  -----------------  --------------------------  ------------
$ sudo isula-build ctr-img tag a24bb4013296 alpine:latest
$ sudo isula-build ctr-img images
----------------------------------------------  -----------  -----------------  --------------------------  ------------
 REPOSITORY                                      TAG          IMAGE ID           CREATED                     SIZE
----------------------------------------------  -----------  -----------------  --------------------------  ------------
 docker.io/library/alpine                        latest       a24bb4013296       2020-05-29 21:19:46         5.85 MB
 localhost/alpine                                latest       a24bb4013296       2020-05-29 21:19:46         5.85 MB
----------------------------------------------  -----------  -----------------  --------------------------  ------------
```

### Authentication of the Remote Image Repository

We can `login` or `logout` an image repository

Login Usage：

`isula-build login dockerhub.io`

We can run the `login` command to login into an image repository

Currently, the following flags are supported:

```bash
Flags:
  -p, --password-stdin    Read password from stdin
  -u, --username string   Username to access registry
```

`cat creds.txt | sudo isula-build login -u cooper -p mydockerhub.io`

```bash
$ sudo isula-build login mydockerhub.io -u cooper
Password:
Login Succeeded
```

Logout Usage:

`isula-build logout mydockerhub.io`

We can run the `logout` command to logout from an image repository

Currently, the following flags are supported:

```bash
Flags:
  -a, --all   Logout all registries
```

#### -a, --all

logout from all registries

```bash
$ sudo isula-build logout -a
Removed authentications
```

### Version query

We can run the `version` command to view the current version information.

```bash
$ sudo isula-build version
Client:
  Version:       0.0.9
  Go Version:    go1.13.3
  Git Commit:    c687e4b
  Built:         Thu Jun 11 19:02:45 2020
  OS/Arch:       linux/amd64

Server:
  Version:       0.0.9
  Go Version:    go1.13.3
  Git Commit:    c687e4b
  Built:         Thu Jun 11 19:02:45 2020
  OS/Arch:       linux/amd64
```

### Show system information

We can use the `info` command to view isula-build system information.

Currently, the following flags are supported:

```bash
Flags:
  -H, --human-readable   print memory info in human readable format, use powers of 1000
```

#### -H, --human-readable

print memory info in human readable format, use powers of 1000

Usage:

`isula-build info -H`

```bash
$ sudo isula-build info -H
General:
  MemTotal:     10.3 GB
  MemFree:      2.24 GB
  SwapTotal:    8.46 GB
  SwapFree:     8.45 GB
  OCI Runtime:  runc
  DataRoot:     /var/lib/isula-build/
  RunRoot:      /var/run/isula-build/
  Builders:     0
  Goroutines:   12
Store:
  Storage Driver:     overlay
  Backing Filesystem: extfs
Registry:
  Search Registries:
    docker.io
  Insecure Registries:
    localhost:5000
```

### Load an image

We can use the `load` command to load an image from the tarfile.

Usage:

`isula-build ctr-img load [flags]`

Currently, the following flags are supported:

```bash
Flags:
  -i, --input string   Path to local tarball
```

Use this command as follow:

```bash
$ sudo isula-build ctr-img load -i image.tar
Getting image source signatures
Copying blob sha256:37841116ad3b1eeea972c75ab8bad05f48f721a7431924bc547fc91c9076c1c8
Copying blob sha256:6eb4c21cc3fcb729a9df230ae522c1d3708ca66e5cf531713dbfa679837aa287
Copying config sha256:76a4dd2d5d6a18323ac8d90f959c3c8562bf592e2a559bab9b462ab600e9e5fc
Writing manifest to image destination
Storing signatures
Loaded image as 76a4dd2d5d6a18323ac8d90f959c3c8562bf592e2a559bab9b462ab600e9e5fc
```
