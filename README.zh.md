# isula-build

isula-build是iSula容器团队推出的容器镜像构建工具，支持通过Dockerfile文件快速构建容器镜像。

isula-build采用服务端/客户端模式，其中`isula-build`为客户端，提供了一组命令行工具，用于镜像构建及管理等；`isula-builder`为服务端，用于处理客户端管理请求，作为守护进程常驻后台。

isula-build提供的命令行工具提供了很多功能，例如：

- 通过`Dockerfile`构建容器镜像（build）
- 查看本地持久化构建镜像（image）
- 导入容器基础镜像（import）
- 导入层叠镜像（load）
- 删除本地持久化镜像（rm）
- 导出层叠镜像（save）
- 给本地持久化镜像打标签（tag）
- 拉取镜像到本地（pull）
- 将本地镜像推送到远程仓库（push）
- 查看运行环境与系统信息（info）
- 登录远端镜像仓库（login）
- 退出远端镜像仓库（logout）
- 版本查询（version）

除此之外，我们提供了以下能力：

- 兼容`Dockerfile`语法
- 支持文件属性扩展，如`IMA`等
- 支持不同种类镜像导出方式，如导出到本地tar包（docker-archive）、iSulad
- ...

## 详细文档

- [中文版](./doc/manual_zh.md)
- [使用教程](./doc/manual_zh.md#使用指南)

## 开始

### 在openEuler上安装

#### 从源码开始编译安装

为了顺利从源码编译，以下包需要被安装在你的操作系统中：

- make
- golang（大于等于1.13版本）
- btrfs-progs-devel
- device-mapper-devel
- glib2-devel
- gpgme-devel
- libassuan-devel
- libseccomp-devel
- git
- bzip2
- systemd-devel

你可以通过`yum`安装这些依赖：

```sh
sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel gpgme-devel libassuan-devel libseccomp-devel git bzip2 systemd-devel golang
```

使用`git`拉取源代码：

```sh
git clone https://gitee.com/openeuler/isula-build.git
```

进入源码目录开始准备编译：

```sh
cd isula-build
sudo make
```

编译成功之后，你可以通过该命令将编译完毕的二进制以及相关配置文件安装到系统中：

```sh
sudo make install
```

#### 通过RPM包安装

`isula-build`目前已经收录在openEuler的官方源中，你可以使用`yum`或者`rpm`安装该包：

##### 使用`yum`

```sh
sudo yum install -y isula-build
```
> **注意：**
>
> 需要先enable repo配置的update部分
> 你可以在[openEuler repo list](https://repo.openeuler.org/)中找到对应的yum源进行安装

##### 使用`rpm`

下载isula-build的rpm包进行安装

```sh
sudo rpm -ivh isula-build-*.rpm
```

### 运行守护进程

#### 以系统服务运行

如果需要使用`systemd`进行管理isula-build，请参考以下步骤：

```sh
sudo install -p -m 640 ./isula-build.service /etc/systemd/system/isula-build.
sudo systemctl enable isula-build
sudo systemctl start isula-build
```

#### 直接运行二进制

你也可以直接运行isula-builder二进制开启服务：

```sh
sudo isula-builder --dataroot="/var/lib/isula-build"
```

### 构建容器镜像

#### 前提

为了正确构建容器镜像，容器运行时`runc`是必要的

你可以通过安装`docker`或者`docker-runc`来获取`runc`二进制

```sh
sudo yum install docker
```

或者

```sh
sudo yum install docker-runc
```

#### 构建镜像

以下是一个简单的例子教你如何去构建一个容器镜像，更多的详细操作可以参考[使用指南](./doc/manual_zh.md#使用指南)

创建一个构建工作目录，编写一个简单的dockerfile：

```dockerfile
FROM alpine:latest
LABEL foo=bar
COPY ./* /home/dir1/
```

在构建工作目录中构建镜像

```sh
$ sudo isula-build ctr-img build -f Dockerfile .
STEP  1: FROM alpine:latest
STEP  2: LABEL foo=bar
STEP  3: COPY ./* /home/dir1/
Getting image source signatures
Copying blob sha256:
e9235582825a2691b1c91a96580e358c99acfd48082cbf1b92fd2ba4a791efc3
Copying blob sha256:
dc3bca97af8b81508c343b13a08493c7809b474dc25986fcbae90c6722201be3
Copying config sha256:
9ec92a8819f9da1b06ea9ff83307ff859af2959b70bfab101f6a325b1a211549
Writing manifest to image destination
Storing signatures
Build success with image id:
9ec92a8819f9da1b06ea9ff83307ff859af2959b70bfab101f6a325b1a211549
```

#### 列出本地镜像

```sh
$ sudo isula-build ctr-img images
-----------------  -----------  ----------------  ----------------------------------------------
    REPOSITORY         TAG          IMAGE ID                       CREATED
------------------  ----------  ----------------  ----------------------------------------------
     <none>          latest      9ec92a8819f9        2020-06-11 07:45:39.265106109 +0000 UTC
```

### 移除镜像

```sh
$ sudo isula-build ctr-img rm 9ec92a8819f9
Deleted: sha256:86567f7a01b04c662a9657aac436e8d63ecebb26da4252abb016d177721fa11b
```

### 与容器引擎集成

详情可见[直接集成容器引擎](./doc/manual_zh.md#直接集成容器引擎)

## 注意事项

约束、限制以及与`docker build`的差别可见[使用注意事项](./doc/manual_zh.md#使用注意事项)

## 如何贡献

我们很高兴能有新的贡献者加入！

在一切开始之前，请签署[CLA协议](https://openeuler.org/en/cla.html)

## 版权

isula-build遵从**Mulan PSL v2**版权协议
