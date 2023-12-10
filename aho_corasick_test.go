package main

import "testing"

func Test_New(t *testing.T) {
	loadConfig()
	globalDict.Load()
	ack := New(globalDict.mdxDict)
	res := ack.Get("jesus")
	t.Logf("%q output: %v", "jesus", res)
}
