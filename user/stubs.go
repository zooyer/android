package user

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

type Group struct {
	Name   string
	Passwd string
	GID    uint
	Mem    []string
}

type Passwd struct {
	Name   string // user's login name
	Passwd string
	UID    uint // numerical user ID
	GID    uint // numerical group ID
	Gecos  string
	Dir    string // initial working directory
	Shell  string // program to use as shell
}

func androidIInfoToPasswd(info AndroidIDInfo) Passwd {
	return Passwd{
		Name:   info.Name,
		Passwd: "",
		UID:    info.Aid,
		GID:    info.Aid,
		Gecos:  "",
		Dir:    "/",
		Shell:  "/system/bin/sh",
	}
}

func androidIInfoToGroup(info AndroidIDInfo) Group {
	return Group{
		Name:   info.Name,
		Passwd: "",
		GID:    info.Aid,
		Mem:    []string{info.Name},
	}
}

func androidIDToPasswd(id uint) *Passwd {
	for _, info := range AndroidIDs {
		if info.Aid == id {
			var pwd = androidIInfoToPasswd(info)
			return &pwd
		}
	}

	return nil
}

func androidNameToPasswd(name string) *Passwd {
	for _, info := range AndroidIDs {
		if info.Name == name {
			var pwd = androidIInfoToPasswd(info)
			return &pwd
		}
	}

	return nil
}

func androidIDToGroup(id uint) *Group {
	for _, info := range AndroidIDs {
		if info.Aid == id {
			var group = androidIInfoToGroup(info)
			return &group
		}
	}

	return nil
}

func androidNameToGroup(name string) *Group {
	for _, info := range AndroidIDs {
		if info.Name == name {
			var group = androidIInfoToGroup(info)
			return &group
		}
	}

	return nil
}

/* translate a user/group name like app_1234 into the
 * corresponding user/group id (AID_APP + 1234)
 * returns 0 and sets errno to ENOENT in case of error
 */
func appIDFromName(name string) (id uint, err error) {
	var i uint64
	if !strings.HasPrefix(name, "app_") || unicode.IsDigit(rune(name[4])) {
		goto FAIL
	}

	if i, err = strconv.ParseUint(name[4:], 10, 32); err != nil {
		return
	}

	i += AidApp

	/* check for overflow and that the value can be
	 * stored in our 32-bit uid_t/gid_t */
	if i < AidApp || uint64(uint(i)) != i {
		goto FAIL
	}

	return uint(i), nil

FAIL:
	return 0, syscall.ENOENT
}

/* translate a uid into the corresponding app_<uid>
 * passwd structure (sets errno to ENOENT on failure)
 */
func appIDToPasswd(uid uint) *Passwd {
	if uid < AidApp {
		return nil
	}

	var pw = new(Passwd)
	pw.Name = fmt.Sprintf("app_%d", uid-AidApp)
	pw.Dir = "/data"
	pw.Shell = "/system/bin/sh"
	pw.UID = uid
	pw.GID = uid

	return pw
}

/* translate a gid into the corresponding app_<gid>
 * group structure (sets errno to ENOENT on failure)
 */
func appIDToGroup(gid uint) *Group {
	if gid < AidApp {
		return nil
	}

	var group = new(Group)
	group.Name = fmt.Sprintf("app_%d", gid-AidApp)
	group.GID = gid
	group.Mem = []string{group.Name}

	return group
}

func Getpwuid(uid uint) (pw *Passwd) {
	if pw = androidIDToPasswd(uid); pw != nil {
		return pw
	}

	return appIDToPasswd(uid)
}

func Getpwnam(name string) (pw *Passwd) {
	if pw = androidNameToPasswd(name); pw != nil {
		return pw
	}

	id, err := appIDFromName(name)
	if err != nil {
		return
	}

	return appIDToPasswd(id)
}

func GetLogin() string {
	if pw := Getpwuid(uint(os.Getuid())); pw != nil {
		return pw.Name
	}

	return ""
}

func GetGrgid(gid uint) *Group {
	if group := androidIDToGroup(gid); group != nil {
		return group
	}

	return appIDToGroup(gid)
}

func Getgrnam(name string) *Group {
	if group := androidNameToGroup(name); group != nil {
		return group
	}

	id, err := appIDFromName(name)
	if err != nil {
		return nil
	}

	return appIDToGroup(id)
}
