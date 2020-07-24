# integration

isula-build can be integrated with isulad or docker to export the built images to container engine store.

## integration with iSulad
Export the successfully built image directly to iSulad

Constraint: isula-build and iSulad are on the same node

Example：

```
$ sudo isula-build ctr-img build -f Dockerfile -o isulad:busybox:2.0
```

Using parameter -o to export built images to isulad, can be queried by `sudo isula images`

## integration with Docker
Export the successfully built image directly to Docker daemon

Constraint: isulad-build and Docker are on the same node

Example：

```
$ sudo isula-build ctr-img build -f Dockerfile -o docker-daemon:busybox:2.0
```

Using parameter -o to export built images to docker, can be queried by `sudo docker images`
