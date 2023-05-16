# ugit

Reimplementing the concepts of `git`
following https://www.leshenko.net/p/ugit/

# To compile the protobuf
protoc --go_out=.  --go_opt=paths=source_relative ./base/ugit.proto
