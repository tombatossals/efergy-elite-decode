# go-efergy-elite-decode
Go program for decoding a RTL-SDR frame read from an Efergy Elite 1.0

# Dependencies

$ go get github.com/garyburd/redigo/redis


# Command line to read the Efergy Elite data
$ rtl_fm -f 433524500 -s 200000 -r 96000 -g 19.7 | go run decode.go
