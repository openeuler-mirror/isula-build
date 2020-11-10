%global is_systemd 1

Name: isula-build
Version: 0.9.4
Release: 2
Summary: A tool to build container images
License: Mulan PSL V2
URL: https://gitee.com/openeuler/isula-build
Source0: https://gitee.com/openeuler/isula-build/repository/archive/v%{version}.tar.gz
Source1: git-commit
Source2: VERSION-openeuler
Source3: apply-patches
Source4: gen-version.sh
Source5: series.conf
Source6: patch.tar.gz
BuildRequires: make btrfs-progs-devel device-mapper-devel glib2-devel gpgme-devel
BuildRequires: libassuan-devel libseccomp-devel git bzip2 go-md2man systemd-devel
BuildRequires: golang >= 1.13
%if 0%{?is_systemd}
BuildRequires: pkgconfig(systemd)
Requires: systemd-units
%endif

%description
isula-build is a tool used for container images building.

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
%{make_build} safe
./bin/isula-build completion > __isula-build

%install
install -d %{buildroot}%{_bindir}
# install binary
install -p -m 555 ./bin/isula-build %{buildroot}%{_bindir}/isula-build
install -p -m 555 ./bin/isula-builder %{buildroot}%{_bindir}/isula-builder
# install service
%if 0%{?is_systemd}
install -d %{buildroot}%{_unitdir}
install -p -m 640 isula-build.service %{buildroot}%{_unitdir}/isula-build.service
%endif
# install config file
install -d -m 750 %{buildroot}%{_sysconfdir}/isula-build
install -p -m 600 ./cmd/daemon/config/configuration.toml %{buildroot}%{_sysconfdir}/isula-build/configuration.toml
install -p -m 600 ./cmd/daemon/config/storage.toml %{buildroot}%{_sysconfdir}/isula-build/storage.toml
install -p -m 600 ./cmd/daemon/config/registries.toml %{buildroot}%{_sysconfdir}/isula-build/registries.toml
install -p -m 400 ./cmd/daemon/config/policy.json %{buildroot}%{_sysconfdir}/isula-build/policy.json
# install bash completion script
install -d %{buildroot}/usr/share/bash-completion/completions
install -p -m 600 __isula-build %{buildroot}/usr/share/bash-completion/completions/isula-build

%clean
rm -rf %{buildroot}

%post
if ! getent group isula > /dev/null; then
    groupadd --system isula
fi

%files
# default perm for files and folder
%defattr(0640,root,root,0550)
%if 0%{?is_systemd}
%config(noreplace) %attr(0640,root,root) %{_unitdir}/isula-build.service
%endif
%attr(555,root,root) %{_bindir}/isula-build
%attr(555,root,root) %{_bindir}/isula-builder
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/configuration.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/storage.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/registries.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/policy.json
/usr/share/bash-completion/completions/isula-build

%changelog
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
