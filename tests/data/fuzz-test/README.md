## How to Construct the go fuzz test

1. make folder the form like `fuzz-test-xxx`
2. put the materials in the folder, they will be use for test script:
```bash
$ tree fuzz-test-builder
   fuzz-test-builder		# test case root dir
   |-- corpus				# dir to store mutation corpus
   |   `-- Dockerfile       # mutation corpus
   |-- Fuzz                 # fuzz go file
   `-- path                 # record relative path to put the Fuzz file
```
3. when the above meterials are ready, go to `isula-build/tests/src`
4. the **ONLY Three Things** you need to do is:
    - copy `TEMPLATE` file to the name you want(*must start with `fuzz_test`*), for example `fuzz_test_xxx.sh`
    - change the variable `test_name` same as the name you just gave
    - uncomment the last line `main "$1"`
5. run the go fuzz shell script by doing `$ bash fuzz_test_xxx.sh`, it will stop fuzzing after 1 minute.
   If you want to change the default run time, you could do like `$ bash fuzz_test_xxx.sh 2h` to keep running 2 hours

## References
All corpus used for fuzzing are collected from the dockerfile of following projects:

- [busybox](https://github.com/docker-library/busybox)
- [distribution-library-image](https://github.com/docker/distribution-library-image)
- [docker-alpine](https://github.com/alpinelinux/docker-alpine)
- [docker-consul](https://github.com/hashicorp/docker-consul)
- [docker-nginx](https://github.com/nginxinc/docker-nginx)
- [docker-node](https://github.com/nodejs/docker-node)
- [docker](https://github.com/docker-library/docker)
- [golang](https://github.com/docker-library/golang)
- [harbor](https://github.com/goharbor/harbor)
- [hello-world](https://github.com/docker-library/hello-world)
- [httpd](https://github.com/docker-library/httpd)
- [influxdata-docker](https://github.com/influxdata/influxdata-docker)
- [mariadb](https://github.com/docker-library/mariadb)
- [memcached](https://github.com/docker-library/memcached)
- [mongo](https://github.com/docker-library/mongo)
- [mysql](https://github.com/docker-library/mysql)
- [openjdk](https://github.com/docker-library/openjdk)
- [postgres](https://github.com/docker-library/postgres)
- [python](https://github.com/docker-library/python)
- [redis](https://github.com/docker-library/redis)
- [traefik-library-image](https://github.com/containous/traefik-library-image)
