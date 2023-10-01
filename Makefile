get_python_versions:
	curl https://raw.githubusercontent.com/actions/python-versions/main/versions-manifest.json | jq -r '.[].version' >./ci/test/python-versions

run_ci_all:
	./run_ci.sh -all

run_ci_setup:
	./run_ci.sh -setup

run_ci_containers:
	./run_ci.sh -containers
