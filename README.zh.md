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

- 📖 [容器镜像构建](./doc/manual_zh.md) - 完整用户手册
- 📚 [使用指南](./doc/manual_zh.md#使用指南) - 详细使用说明
- 🔧 [集成指南](./doc/manual_zh.md#直接集成容器引擎) - 与容器引擎集成

## 开始

### 快速链接

- 📖 [用户手册](./doc/manual_zh.md) - 完整文档
- 🛠️ [构建指南](#在openeuler上安装) - 安装说明
- 🔗 [集成](./doc/manual_zh.md#直接集成容器引擎) - 容器引擎集成
- ⚠️ [注意事项](./doc/manual_zh.md#使用注意事项) - 重要提示和限制

### 开发环境设置

#### 前置要求

- **操作系统**: Linux（推荐：openEuler、CentOS、Ubuntu）
- **Go**: 1.15 或更高版本
- **构建工具**: make, git
- **依赖项**: 请参考下面的安装部分

#### 快速设置

1. **克隆仓库:**
   ```bash
   git clone https://gitee.com/openeuler/isula-build.git
   cd isula-build
   ```

2. **安装依赖 (openEuler):**
   ```bash
   sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel \
       gpgme-devel libassuan-devel libseccomp-devel git bzip2 \
       go-md2man systemd-devel golang
   ```

3. **从源码构建:**
   ```bash
   sudo make all
   sudo make install
   ```

其他发行版的安装方法，请参考下面的安装部分。

#### IDE 配置

**推荐的 IDE:**
- **GoLand/IntelliJ IDEA**: 直接打开项目，Go 插件会自动配置
- **VSCode**: 安装 Go 扩展，使用 `go.toolsEnvVars` 配置
- **Vim/Neovim**: 使用 gopls（Go 语言服务器）配合 vim-go 或 coc.nvim

**Go 开发环境设置:**
```bash
# 安装 Go 工具
go install golang.org/x/tools/gopls@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 验证 Go 环境
go env
```

**代码质量工具:**
- 静态分析: `hack/static_check.sh all`（全量检查）或 `hack/static_check.sh last`（最新提交）
- 使用 `golangci-lint`，配置文件为 `.golangci.yml`
- 运行测试: `make test` 或 `go test ./...`

### 在openEuler上安装

#### 从源码开始编译安装

为了顺利从源码编译，以下包需要被安装在用户的操作系统中：

- make
- golang（大于等于1.15版本）
- btrfs-progs-devel
- device-mapper-devel
- glib2-devel
- gpgme-devel
- libassuan-devel
- libseccomp-devel
- git
- bzip2
- systemd-devel

用户可以通过`yum`安装这些依赖：

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
执行该命令后可以看到"Build"下面有5个target可编译：
- all
- isula-build
- isula-builder
- safe
- debug

其中`sudo make all`编译客户端isula-build和服务端isula-builder，`sudo make isula-build`仅编译客户端，`sudo make isula-builder`仅编译服务端，`sudo make safe`编译客户端和服务端生成安全编译的二进制文件，`sudo make debug`编译客户端和服务端生成带有debug的二进制文件，用户可以根据需求选择编译具体的target

编译成功之后，用户可以通过该命令将编译完毕的二进制以及相关配置文件安装到系统中：

```sh
sudo make install
```

#### 通过RPM包安装

`isula-build`目前已经收录在openEuler的官方源中，用户可以使用`yum`或者`rpm`安装该包：

##### 使用`yum`

```sh
sudo yum install -y isula-build
```
> **注意：**
>
> 需要先enable repo配置的update部分
> 用户可以在[openEuler repo list](https://repo.openeuler.org/)中找到对应的yum源进行安装

##### 使用`rpm`

下载isula-build的rpm包进行安装

```sh
sudo rpm -ivh isula-build-*.rpm
```

### 运行守护进程

#### 以系统服务运行（推荐）

如果需要使用`systemd`进行管理isula-build，请参考以下步骤：

```bash
# 安装 systemd 服务文件
sudo install -p -m 640 ./isula-build.service /etc/systemd/system/isula-build.service

# 启用并启动服务
sudo systemctl enable isula-build
sudo systemctl start isula-build

# 检查服务状态
sudo systemctl status isula-build
```

#### 直接运行二进制

用户也可以直接运行isula-builder二进制开启服务：

```bash
sudo isula-builder --dataroot="/var/lib/isula-build"
```

**注意**: 直接运行时，如果服务正在以 systemd 方式运行，请先停止：
```bash
sudo systemctl stop isula-build
```

### 构建容器镜像

#### 前提

为了正确构建容器镜像，容器运行时`runc`是必要的

用户可以通过安装`docker`或者`docker-runc`来获取`runc`二进制

```sh
sudo yum install docker
```

或者

```sh
sudo yum install docker-runc
```

#### 构建镜像

以下是一个简单的例子教用户如何去构建一个容器镜像，更多的详细操作可以参考[使用指南](./doc/manual_zh.md#使用指南)

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

## 故障排查

### 常见问题

#### 构建问题

**问题**: `go: command not found`
```bash
# 解决方案: 安装 Go（1.15 或更高版本）
# 在 openEuler 上:
sudo yum install golang

# 或从 https://golang.org/dl/ 下载
```

**问题**: 构建时缺少依赖
```bash
# 解决方案: 安装所有必需的开发包
sudo yum install make btrfs-progs-devel device-mapper-devel glib2-devel \
    gpgme-devel libassuan-devel libseccomp-devel git bzip2 \
    go-md2man systemd-devel golang
```

**问题**: 构建时权限错误
```bash
# 解决方案: 确保您有 sudo 权限进行安装
# 某些构建步骤需要 root 权限
sudo make install
```

#### 运行时问题

**问题**: 无法连接到 isula-builder 守护进程
```bash
# 解决方案: 检查守护进程是否运行
sudo systemctl status isula-build

# 或检查进程是否运行
ps aux | grep isula-builder

# 如果未运行，启动守护进程
sudo systemctl start isula-build
```

**问题**: 构建镜像时权限被拒绝
```bash
# 解决方案: 确保您有适当的权限
# isula-build 通常需要 sudo 来构建
sudo isula-build ctr-img build -f Dockerfile .
```

**问题**: 构建时找不到 `runc`
```bash
# 解决方案: 安装 runc 或 docker
sudo yum install docker
# 或
sudo yum install docker-runc
```

#### 开发问题

**问题**: 静态检查失败
```bash
# 解决方案: 安装 golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行静态检查
hack/static_check.sh all
```

**问题**: 测试失败
```bash
# 解决方案: 确保所有依赖都已安装
# 使用详细输出运行测试
go test -v ./...

# 或使用 make
make test
```

更多详细的故障排查信息，请参考：
- [用户手册](./doc/manual_zh.md)
- [使用注意事项](./doc/manual_zh.md#使用注意事项)

## 如何贡献

我们欢迎您的贡献！以下是如何开始：

### 贡献前准备

1. **签署 CLA**: 在贡献之前，请签署 [CLA协议](https://openeuler.org/en/cla.html)
2. **阅读文档**: 熟悉[用户手册](./doc/manual_zh.md)
3. **查看现有问题**: 查找现有问题或创建新问题

### 开发工作流程

1. **Fork 仓库**并克隆您的 fork
2. **创建特性分支**: `git checkout -b feature/您的特性名称`
3. **进行更改**并确保通过：
   - 静态检查: `hack/static_check.sh all`
   - 测试: `make test` 或 `go test ./...`
4. **提交更改**: 使用清晰、描述性的提交信息
5. **推送到您的 fork**并创建 Pull Request

### 代码质量标准

- 遵循 Go 编码标准和约定
- 提交前运行 `hack/static_check.sh all`
- 确保所有测试通过
- 如需要，更新文档
- 为新功能添加测试

### 获取帮助

- 查看[文档](./doc/manual_zh.md)
- 查看现有问题和 PR
- 在问题或讨论中提问

## 版权

isula-build遵从**Mulan PSL v2**版权协议

更多信息，请查看 [LICENSE](./License/LICENSE) 文件。
