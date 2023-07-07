%global is_systemd 1

Name: isula-build
Version: 0.9.6
Release: 18
Summary: A tool to build container images
License: Mulan PSL V2
URL: https://gitee.com/openeuler/isula-build
Source0: https://gitee.com/openeuler/isula-build/repository/archive/v%{version}.tar.gz
Source1: git-commit
Source2: VERSION-vendor
Source3: apply-patches
Source4: gen-version.sh
Source5: series.conf
Source6: patch.tar.gz

BuildRequires: make btrfs-progs-devel device-mapper-devel glib2-devel gpgme-devel
BuildRequires: libassuan-devel libseccomp-devel git bzip2 systemd-devel
BuildRequires: golang >= 1.13
%if 0%{?is_systemd}
BuildRequires: pkgconfig(systemd)
Requires: systemd-units
%endif

%description
isula-build is a tool used for container images building.

%debug_package

%prep
cp %{SOURCE0} .
cp %{SOURCE1} .
cp %{SOURCE2} .
cp %{SOURCE3} .
cp %{SOURCE4} .
cp %{SOURCE5} .
cp %{SOURCE6} .

%build
sh ./apply-patches
%if "toolchain"=="clang"
patch -p1<patch/0137-fix-clang.patch
%endif
%{make_build} safe
./bin/isula-build completion > __isula-build

%install
install -d %{buildroot}%{_bindir}
# install binary
install -p ./bin/isula-build %{buildroot}%{_bindir}/isula-build
install -p ./bin/isula-builder %{buildroot}%{_bindir}/isula-builder
# install service
%if 0%{?is_systemd}
install -d %{buildroot}%{_unitdir}
install -p isula-build.service %{buildroot}%{_unitdir}/isula-build.service
%endif
# install config file
install -d %{buildroot}%{_sysconfdir}/isula-build
install -p ./cmd/daemon/config/configuration.toml %{buildroot}%{_sysconfdir}/isula-build/configuration.toml
install -p ./cmd/daemon/config/storage.toml %{buildroot}%{_sysconfdir}/isula-build/storage.toml
install -p ./cmd/daemon/config/registries.toml %{buildroot}%{_sysconfdir}/isula-build/registries.toml
install -p ./cmd/daemon/config/policy.json %{buildroot}%{_sysconfdir}/isula-build/policy.json
# install bash completion script
install -d %{buildroot}/usr/share/bash-completion/completions
install -p -m 600 __isula-build %{buildroot}/usr/share/bash-completion/completions/isula-build

%clean
rm -rf %{buildroot}

%pre
if ! getent group isula > /dev/null; then
    groupadd --system isula
fi

%files
# default perm for files and folder
%defattr(0640,root,root,0550)
%if 0%{?is_systemd}
%config(noreplace) %attr(0640,root,root) %{_unitdir}/isula-build.service
%endif
%attr(550,root,isula) %{_bindir}/isula-build
%attr(550,root,root) %{_bindir}/isula-builder

%dir %attr(650,root,root) %{_sysconfdir}/isula-build
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/configuration.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/storage.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/registries.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/policy.json
/usr/share/bash-completion/completions/isula-build

%changelog
* Fri Jul 07 2023 zhangxiang <zhangxiang@iscas.ac.cn> - 0.9.6-18
- Type:bugfix
- CVE:NA
- SUG:NA
- DESC:fix clang build error

* Thu Feb 02 2023 daisicheng <daisicheng@huawei.com> - 0.9.6-17
- Type:bugfix
- CVE:NA
- SUG:NA
- DESC:add manifest.json verification before loading a tar

* Thu Dec 22 2022 daisicheng <daisicheng@huawei.com> - 0.9.6-16
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:add some DT tests

* Wed Dec 07 2022 jingxiaolu <lujingxiao@huawei.com> - 0.9.6-15
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:add read lock in load/import/pull to fix GC preempts to exit subprocess

* Wed Nov 23 2022 Lixiang <cooper.li@huawei.com> - 0.9.6-14
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:use vendor instead specific vendor name

* Tue Nov 01 2022 daisicheng <daisicheng@huawei.com> - 0.9.6-13
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:fix the problem that the /var/lib/isula-build/storage/overlay is still existed when killing daemon

* Wed Sep 14 2022 xingweizheng <xingweizheng@huawei.com> - 0.9.6-12
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:improve security compile option of isula-build binary

* Fri Aug 19 2022 daisicheng <daisicheng@huawei.com> - 0.9.6-11
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:modify the Makefile and README document;add the constraints and limitations of the doc;fix the possible file leakage problem in util/cipher

* Tue Jul 26 2022 lujingxiao <lujingxiao@huawei.com> - 0.9.6-10
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:registries.toml could not be empty;hosts, resolv.conf, .dockerignore file could be empty

* Tue Jul 26 2022 xingweizheng <xingweizheng@huawei.com> - 0.9.6-9
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:supplement patches in series.conf

* Wed Jun 15 2022 xingweizheng <xingweizheng@huawei.com> - 0.9.6-8
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:sync upstream patches

* Thu May 26 2022 loong_C <loong_c@yeah.net> - 0.9.6-7
- fix spec changelog date

* Wed Mar 16 2022 xingweizheng <xingweizheng@huawei.com> - 0.9.6-6
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:disable go-selinux on openEuler

* Thu Jan 13 2022 DCCooper <1866858@gmail.com> - 0.9.6-5
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:add syscall "statx" in seccomp

* Fri Dec 31 2021 jingxiaolu <lujingxiao@huawei.com> - 0.9.6-4
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:refactor image separator related

* Thu Dec 23 2021 DCCooper <1866858@gmail.com> - 0.9.6-3
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:sync upstream patches

* Wed Dec 08 2021 DCCooper <1866858@gmail.com> - 0.9.6-2
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:sync upstream patch

* Mon Nov 29 2021 DCCooper <1866858@gmail.com> - 0.9.6-1
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:Bump version to 0.9.6

* Wed Nov 17 2021 jingxiaolu <lujingxiao@huawei.com> - 0.9.5-21
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:add repo to local image when output transporter is docker://

* Wed Nov 10 2021 DCCooper <1866858@gmail.com> - 0.9.5-20
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:add log info for layers processing

* Thu Nov 04 2021 DCCooper <1866858@gmail.com> - 0.9.5-19
- Type:bugfix
- CVE:NA
- SUG:restat
- DESC:fix panic when using image ID to save separated image

* Wed Nov 03 2021 lixiang <lixiang172@huawei.com> - 0.9.5-18
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:fix loaded images cover existing images name and tag

* Wed Nov 03 2021 DCCooper <1866858@gmail.com> - 0.9.5-17
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:optimize function IsExist and add more testcase for filepath.go

* Wed Nov 03 2021 DCCooper <1866858@gmail.com> - 0.9.5-16
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:fix random sequence for saving separated image tarball

* Tue Nov 02 2021 lixiang <lixiang172@huawei.com> - 0.9.5-15
- Type:requirement
- CVE:NA
- SUG:restart
- DESC:support save/load separated image, add relative test cases and bugfixes as well

* Mon Oct 25 2021 DCCooper <1866858@gmail.com> - 0.9.5-14
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:sync patches from upstream, including relocate export package, clean code for tests and golint

* Thu Oct 14 2021 DCCooper <1866858@gmail.com> - 0.9.5-13
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:use pre instead of pretrans for groupadd

* Fri Sep 03 2021 xingweizheng <xingweizheng@huawei.com> - 0.9.5-12
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:fix for save single image with multiple tags when id first

* Tue Aug 31 2021 jingxiaolu <lujingxiao@huawei.com> - 0.9.5-11
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:sync patches from upstream, including fix for save multiple tags, test cases improvement

* Mon Jul 26 2021 DCCooper <1866858@gmail.com> - 0.9.5-10
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:update documents about file mode

* Mon Jul 26 2021 DCCooper <1866858@gmail.com> - 0.9.5-9
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:modify file mode for isula-build client binary and public key

* Wed Jun 16 2021 DCCooper <1866858@gmail.com> - 0.9.5-8
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:sync patch from upstream

* Wed Jun 02 2021 DCCooper <1866858@gmail.com> - 0.9.5-7
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:sync patches from upstream

* Wed Mar 03 2021 lixiang <lixiang172@huawei.com> - 0.9.5-6
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:sync patches from upstream

* Wed Feb 10 2021 lixiang <lixiang172@huawei.com> - 0.9.5-5
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:remove empty lines when showing image list

* Tue Feb 09 2021 DCCooper <1866858@gmail.com> - 0.9.5-4
- Type:bugfix
- CVE:NA
- SUG:restart
- DESC:remove Healthcheck field when build from scratch

* Tue Feb 09 2021 DCCooper <1866858@gmail.com> - 0.9.5-3
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:remove go-md2man build require

* Thu Feb 4 2021 leizhongkai<leizhongkai@huawei.com> - 0.9.5-2
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:make `isula-build ctr-img images` display comfortably

* Tue Jan 26 2021 lixiang <lixiang172@huawei.com> - 0.9.5-1
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:Bump version to 0.9.5

* Fri Dec 11 2020 lixiang <lixiang172@huawei.com> - 0.9.4-14
- Type:enhancement
- CVE:NA
- SUG:restart
- DESC:Modify gen-version script and add changelog automatically

* Fri Dec 11 2020 lujingxiao <lujingxiao@huawei.com> - 0.9.4-13
- Change default umask of isula-builder process

* Tue Dec 08 2020 caihaomin<caihaomin@huawei.com> - 0.9.4-12
- Fix printing FROM command double times to console

* Tue Dec 08 2020 caihaomin<caihaomin@huawei.com> - 0.9.4-11
- Fix problems found by code review

* Tue Dec 08 2020 caihaomin<caihaomin@huawei.com> - 0.9.4-10
- Add more fuzz tests

* Tue Dec 08 2020 caihaomin<caihaomin@huawei.com> - 0.9.4-9
- Imporve daemon push and pull unit test

* Fri Nov 27 2020 lixiang <lixiang172@huawei.com> - 0.9.4-8
- Add compile flag ftrapv and enable debuginfo

* Thu Nov 20 2020 xiadanni <xiadanni1@huawei.com> - 0.9.4-7
- Mask /proc/pin_memory

* Thu Nov 19 2020 lixiang <lixiang172@huawei.com> - 0.9.4-6
- Support build Dockerfile only have FROM command

* Wed Nov 18 2020 lixiang <lixiang172@huawei.com> - 0.9.4-5
- Delete patches no longer usefull

* Tue Nov 17 2020 lixiang <lixiang172@huawei.com> - 0.9.4-4
- Fix unsuitable filemode for isula-build(er)

* Thu Nov 12 2020 lixiang <lixiang172@huawei.com> - 0.9.4-3
- Chown config root path before daemon started

* Tue Nov 10 2020 lixiang <lixiang172@huawei.com> - 0.9.4-2
- Fix panic when user knock ctrl-c in pull/push/save command

* Fri Nov 06 2020 lixiang <lixiang172@huawei.com> - 0.9.4-1
- Bump version to 0.9.4

* Thu Sep 10 2020 lixiang <lixiang172@huawei.com> - 0.9.3-2
- Sync patch from upstream

* Thu Sep 10 2020 lixiang <lixiang172@huawei.com> - 0.9.3-1
- Bump version to 0.9.3

* Fri Sep 04 2020 lixiang <lixiang172@huawei.com> - 0.9.2-3
- Fix Source0 and do not startup after install by default

* Sat Aug 15 2020 lixiang <lixiang172@huawei.com> - 0.9.2-2
- Add bash completion script in rpm

* Wed Aug 12 2020 xiadanni <xiadanni1@huawei.com> - 0.9.2-1
- Bump version to 0.9.2

* Wed Aug 05 2020 xiadanni <xiadanni1@huawei.com> - 0.9.1-1
- Bump version to 0.9.1

* Sat Jul 25 2020 lixiang <lixiang172@huawei.com> - 0.9.0-1
- Package init
