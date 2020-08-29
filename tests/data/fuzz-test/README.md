## How to Construct the go fuzz test

1. make folder the form like `fuzz-test-xxx`
2. put the materials in the folder, they will be use for test script:
```bash
$ tree fuzz-test-builder
   fuzz-test-builder		# test case root dir
   |-- corpus				# dir to store mutation corpus
   |   `-- Dockerfile       # mutation corpus
   |-- Fuzz.go              # fuzz go file
   `-- path                 # record relative path to put the Fuzz.go
```
3. when the above meterials are ready, go to `isula-build/tests/src`
4. the **ONLY Three Things** you need to do is:
    - copy `fuzz-test-template.sh` to the name you want(*must start with `fuzz-test`*), for example `fuzz-test-xxx.sh`
    - change the variable `test_name` same as the name you just gave
    - uncomment the last line `main "$1"`
5. run the go fuzz shell script by doing `$ bash fuzz-test-xxx.sh`, it will stop fuzzing after 1 minute.
   If you want to change the default run time, you could do like `$ bash fuzz-test-xxx.sh 2h` to keep running 2 hours
