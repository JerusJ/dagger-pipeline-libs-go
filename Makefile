test_ui:
	dagger version
	dagger run go run .

test:
	go run . -containers
