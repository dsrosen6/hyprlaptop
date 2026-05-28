run:
	go run main.go listen

install:
	go build -o ./bin/hyprdocked

cp-path:
	sudo cp ./bin/hyprdocked /usr/bin/hyprdocked

cp-service:
	sudo cp ./systemd/hyprdocked.service /usr/lib/systemd/user

svc-daemon-reload:
	systemctl --user daemon-reload

stop-service:
	systemctl --user stop hyprdocked.service

start-service: svc-daemon-reload
	systemctl --user start hyprdocked.service

restart-service: svc-daemon-reload
	systemctl --user restart hyprdocked.service

enable-service: svc-daemon-reload
	systemctl --user enable hyprdocked.service --now

disable-service: svc-daemon-reload
	systemctl --user disable hyprdocked.service

all-install: install cp-path cp-service svc-daemon-reload

tail-log: 
	tail -f ~/dock.log
