package main

import (
	"os"
)

func main() {
	rootCmd, _ := NewRootCmd()
	if err := Execute(rootCmd); err != nil {
		// TODO: enable this for 0.41
		//switch e := err.(type) {
		//case server.ErrorCode:
		//	os.Exit(e.Code)
		//default:
		os.Exit(1)
		//}
	}
}
