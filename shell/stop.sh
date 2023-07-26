# まず`chmod +x shell/stop.sh`してください。
pkill -f 'go run cmd/myapp/main.go'
unset MY_SIGNING_KEY