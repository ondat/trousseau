%:
	mkdir -p bin && test -f ./bin/task || (cd bin ; curl -Ls https://github.com/go-task/task/releases/download/v3.13.0/task_linux_amd64.tar.gz | tar -xz task)

	./bin/task $@