cd app/cache-server/
go build -o ../../bin/server.bin
cd ../../
./bin/server.bin -conf=./conf/server.conf