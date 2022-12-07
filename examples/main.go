package main

import (
	"fmt"
	"time"

	"github.com/mr-karan/barreldb/pkg/barrel"
)

func main() {
	barrel, err := barrel.Init(barrel.Opts{
		Dir:               ".",
		ReadOnly:          false,
		EnableFSync:       true,
		MaxActiveFileSize: 1 << 4,
	})
	if err != nil {
		panic(err)
	}

	if err := barrel.PutEx("hello", []byte("world"), time.Second*5); err != nil {
		panic(err)
	}
	if err := barrel.Put("good", []byte("bye")); err != nil {
		panic(err)
	}

	val, err := barrel.Get("hello")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(val))

	val, err = barrel.Get("good")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(val))

	keys := barrel.List()
	fmt.Println(keys)

	val, err = barrel.Get("hello")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(val))

	barrel.Shutdown()
}
