cd app/cache-proxy/
go build -o ../../bin/proxy.bin
cd ../../
./bin/proxy.bin -conf=./conf/proxy.conf