run:
	go run main.go listen


cp-service:
	sudo cp ./systemd/hyprdocked.service /usr/lib/systemd/user

install:
	go build -o ./bin/hyprdocked

cp-path:
	sudo cp ./bin/hyprdocked /usr/bin/hyprdocked

stop-service:
	systemctl --user stop hyprdocked.service

start-service:
	systemctl --user start hyprdocked.service

restart-service:
	systemctl --user restart hyprdocked.service

enable-service:
	systemctl --user enable hyprdocked.service --now

disable-service:
	systemctl --user disable hyprdocked.service

all-install: install cp-path cp-service
