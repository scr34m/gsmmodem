module github.com/scr34m/gsmmodem

go 1.20

require (
	github.com/tarm/serial v0.0.0-20180830185346-98f6abe2eb07
	github.com/xlab/at v0.0.0-20220814165740-379970a8a2cb
)

require golang.org/x/sys v0.22.0 // indirect

replace github.com/xlab/at => github.com/scr34m/at v0.0.0-20240729144455-546b88c11bf8
