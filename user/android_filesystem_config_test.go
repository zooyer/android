package user

import "testing"

func TestAndroidIDCount(t *testing.T) {
	t.Log(AndroidIDCount())
}

func TestFSConfig(t *testing.T) {
	for _, dir := range AndroidDirs {
		t.Log(FSConfig(dir.Prefix, true))
	}
	for _, file := range AndroidFiles {
		t.Log(FSConfig(file.Prefix, false))
	}
}
