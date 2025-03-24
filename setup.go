package main

import (
	_ "embed"
	"os/user"
	"runtime"
)

func isRoot() (bool, error) {
	u, err := user.Current()
	if err != nil {
		return false, err
	}
	if runtime.GOOS != "windows" {
		return u.Uid == "0", nil
	}
	ids, err := u.GroupIds()
	if err != nil {
		return false, err
	}
	for i := range ids {
		if ids[i] == "S-1-5-32-544" { // SID for the built-in Administrators group
			return true, nil
		}
	}
	return false, nil
}
