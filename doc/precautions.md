# Precautions
This chapter is something about constraints, limitations and differences with `docker build` when use isula-builder build images.

## Constraints or Limitations
1. When export an image to [`iSulad`](https://gitee.com/openeuler/iSulad/blob/master/README.md), a tag is necessary.
2. Because oci runtime binary will be called by `isula-builder` when executing `RUN` command, the integrity of the runtime binary should be guaranteed by the user.
3. DataRoot should not be set in tmpfs.
4. `Overlay2` is the only storage driver supported by isula-builder currently.
5. Docker image is the only image format supported by isula-builder currently.
6. File permission of Dockerfile is strongly recommended to restrict as 0600, avoiding tampering by other users.
7. Only host network is supported by `RUN` command currently.
8. When export image to a tarball, only `tar.gz` compression format supported by isula-builder currently.


##  Differences with `docker build`
The `isula-build` compatible with [Dockerfile specification](https://docs.docker.com/engine/reference/builder), but there are also some subtle differences between `isula-builder` and `docker build` are as follows:
1. Commit every build stage, but not every line.
2. Build cache is not supported by isula-builder.
3. Only `RUN` command will be executed in the build container.
4. Build history is not supported currently.
5. Stage name can be start with a number.
6. The length of the stage name is limited to 64 in `isula-builder`.
7. `ADD` command's source can not support remote URL currently.
8. Not support resource quota for a single build request, but you can limit the `isula-builder` instead.
9. `isula-builder` add each origin layer tar size to get the image size, but docker only uses the diff content of each layer. So the image size listed by `isula-builder images` is a little different.
