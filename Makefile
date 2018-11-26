all:im_serviceTest

im_serviceTest: im_serviceTest.go client.go  message.go protocol.go route_message.go
	go build im_serviceTest.go client.go  message.go protocol.go route_message.go
