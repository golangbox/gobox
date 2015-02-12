GOBOX_PATH=$(pwd)/../test
CLIENT_PATH=$(pwd)/client.go

echo $GOBOX_PATH

go run $CLIENT_PATH $GOBOX_PATH &

CLIENT_PROCESS=$!

echo $CLIENT_PROCESS

cd $GOBOX_PATH

touch f1
echo "foo" >> f1
rm f1

kill $CLIENT_PROCESS








