FROM alpine AS uuid
COPY uuid /src/uuid

FROM alpine AS data
COPY data /src/data

FROM alpine AS foo
ARG USED_ARG="used_value"
RUN echo ${USED_ARG} > /root/value
RUN echo ${USED_ARG}
RUN sleep 5
COPY --from=uuid /src/uuid /root/uuid
COPY --from=data /src/data /root/data
ONBUILD RUN mkdir /home/onbuild

FROM foo
ENV http_proxy=xxxproxy
EXPOSE 80
