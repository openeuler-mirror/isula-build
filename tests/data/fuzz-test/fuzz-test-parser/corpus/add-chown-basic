FROM alpine AS cho
ADD --chown=2367:3267 addchown.txt /tmp
RUN stat -c "user:%u group:%g" /tmp/addchown.txt
WORKDIR /home/workdir
ARG foodir="foo"
COPY addchown.txt /root/${foodir}/foo
ADD addchown.txt /var
ADD ./* /home/tmp/
COPY --chown=root:bin testDir/*.txt /home/new/
COPY --chown=root:1 testDir/b.tx* /home/new/
ADD --chown=0:mail ./addchown.txt .
ADD --chown=0:0 ./testDir/a.txt ./

FROM cho
MAINTAINER isula "isula@huawei.com"
WORKDIR /tmp
CMD ls
VOLUME ["/var/lib/"]
