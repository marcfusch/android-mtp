module github.com/marcfusch/android-mtp

go 1.26.0

require (
	github.com/hanwen/go-fuse/v2 v2.11.0
	github.com/hanwen/go-mtpfs v1.0.1
	github.com/hanwen/usb v0.0.0-20141217151552-69aee4530ac7
	golang.org/x/text v0.44.0
)

require golang.org/x/sys v0.28.0 // indirect

replace github.com/hanwen/go-mtpfs => ./third_party/go-mtpfs
