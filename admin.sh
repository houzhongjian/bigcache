export GIN_MODE=release
cd app/cache-admin/
go build -o ../../bin/admin.bin
cp -r ./web ../../bin
cp -r ./static ../../bin
cd ../../bin
./admin.bin -conf=../conf/admin.conf