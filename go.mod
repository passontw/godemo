module godemo

go 1.24.1

replace go.uber.org/fx => git.trevi.cc/jenkins_deploy/go_fx v0.0.0-20250417091632-df283660ff5e

require go.uber.org/fx v1.24.0

require (
	git.trevi.cc/server/shared_utils v0.2.0 // indirect
	go.uber.org/dig v1.19.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
)
