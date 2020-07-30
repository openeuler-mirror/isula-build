%global is_systemd 1
%global debug_package %{nil}

Name: isula-build
Version: 0.9.0
Release: 1
Summary: A tool to build container images
License: Mulan PSL V2
URL: https://gitee.com/openeuler/isula-build
Source0: isula-build-v0.9.0.tar.gz
Source1: git-commit 
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
%autosetup -n %{name}

%build
cp %{SOURCE1} .
%{make_build} safe

%install
install -d %{buildroot}%{_bindir}
# install binary
install -p -m 550 ./bin/isula-build %{buildroot}%{_bindir}/isula-build
install -p -m 550 ./bin/isula-builder %{buildroot}%{_bindir}/isula-builder
# install service
%if 0%{?is_systemd}
install -d %{buildroot}%{_unitdir}
install -p -m 640 isula-build.service %{buildroot}%{_unitdir}/isula-build.service
%endif
# install config file
install -d %{buildroot}%{_sysconfdir}/isula-build
install -p -m 600 ./cmd/daemon/config/configuration.toml %{buildroot}%{_sysconfdir}/isula-build/configuration.toml
install -p -m 600 ./cmd/daemon/config/storage.toml %{buildroot}%{_sysconfdir}/isula-build/storage.toml
install -p -m 600 ./cmd/daemon/config/registries.toml %{buildroot}%{_sysconfdir}/isula-build/registries.toml
install -p -m 600 ./cmd/daemon/config/policy.json %{buildroot}%{_sysconfdir}/isula-build/policy.json

%clean
rm -rf %{buildroot}

%post
%if 0%{?is_systemd}
systemctl start isula-build
%endif

%preun
%if 0%{?is_systemd}
%systemd_preun isula-build
%endif

%postun
%if 0%{?is_systemd}
%systemd_postun_with_restart isula-build
%endif

%files
# default perm for files and folder
%defattr(0640,root,root,0550)
%if 0%{?is_systemd}
%config(noreplace) %attr(0640,root,root) %{_unitdir}/isula-build.service
%endif
%attr(550,root,root) %{_bindir}/isula-build
%attr(550,root,root) %{_bindir}/isula-builder
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/configuration.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/storage.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/registries.toml
%config(noreplace) %attr(0600,root,root) %{_sysconfdir}/isula-build/policy.json

%changelog
* Sat Jul 25 2020 lixiang <lixiang172@huawei.com> - 0.9.0-1
- Package init
