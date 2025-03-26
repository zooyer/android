/**
 * @Author: zzy
 * @Email: zhangzhongyuan@didiglobal.com
 * @Description:
 * @File: main.go
 * @Package: su
 * @Version: 1.0.0
 * @Date: 2022/10/10 09:25
 */

package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/zooyer/android/user"
)

func help() {
	fmt.Println("usage: su [--path,-p] [UID[,GID[,GID2]...]] [COMMAND [ARG...]]")
	fmt.Println()
	fmt.Println("Switch to WHO (default 'root') and run the given command (default sh).")
	fmt.Println()
	fmt.Println("where WHO is a comma-separated list of user, group,")
	fmt.Println("and supplementary groups in that order.")
	fmt.Println()
}

func help13() {
	fmt.Println("usage: su [--path,-p] [WHO [COMMAND...]]")
	fmt.Println()
	fmt.Println("Switch to WHO (default 'root') and run the given COMMAND (default sh).")
	fmt.Println()
	fmt.Println("WHO is a comma-separated list of user, group, and supplementary groups")
	fmt.Println("in that order.")
	fmt.Println()
}

func execCommand(cmd string) (output string) {
	if cmd == "" {
		return
	}

	const shell = "/system/bin/sh"

	data, _ := exec.Command(shell, "-c", cmd).CombinedOutput()

	return strings.TrimSpace(string(data))
}

func pwtoidCommand(name string) (uid, gid int) {
	output := execCommand(fmt.Sprintf("id -u %s", name))
	uid, _ = strconv.Atoi(output)

	output = execCommand(fmt.Sprintf("id -g %s", name))
	gid, _ = strconv.Atoi(output)

	return
}

func getpwuidCommand() string {
	return execCommand("id -u -n")
}

var paths = [][]string{
	// Android 11 - 13
	{"/product/bin", "/apex/com.android.runtime/bin", "/apex/com.android.art/bin", "/system_ext/bin", "/system/bin", "/system/xbin", "/odm/bin", "/vendor/bin", "/vendor/xbin"},
	// Android 10
	{"/sbin", "/system/sbin", "/product/bin", "/apex/com.android.runtime/bin", "/system/bin", "/system/xbin", "/odm/bin", "/vendor/bin", "/vendor/xbin"},
	// Android 9
	{"/sbin", "/system/sbin", "/system/bin", "/system/xbin", "/odm/bin", "/vendor/bin", "/vendor/xbin"},
	// Android 8
	{"/sbin", "/system/sbin", "/system/bin", "/system/xbin", "/vendor/bin", "/vendor/xbin"},
	// Android 6 - 7
	{"/sbin", "/vendor/bin", "/system/sbin", "/system/bin", "/system/xbin"},
	// Android 4 - 5
	{"/usr/bin", "/bin", "/usr/sbin", "/sbin"},
}

func defPath() string {
	var matched = make(map[string]bool)
	for _, path := range paths {
		var match = true
		for _, folder := range path {
			if _, err := os.Stat(folder); err != nil {
				match = false
			} else {
				matched[folder] = true
			}
		}
		if match {
			return strings.Join(path, ":")
		}
	}

	var list = make([]string, 0, len(matched))
	for path := range matched {
		list = append(list, path)
	}

	return strings.Join(list, ":")
}

func errorExit(status int, err error, msg string) {
	_, _ = fmt.Fprintln(os.Stderr, msg)

	var errno syscall.Errno
	if errors.As(err, &errno) && errno != 0 {
		_, _ = fmt.Fprintln(os.Stderr, errno.Error())
	}

	os.Exit(status)
}

func pwtoid(tok string) (uid, gid int) {
	if pw := user.Getpwnam(tok); pw != nil {
		uid, gid = int(pw.UID), int(pw.GID)
	} else {
		tid, err := strconv.ParseUint(tok, 10, 32)
		if err != nil {
			errorExit(1, err, fmt.Sprintf("invalid uid/gid '%s'", tok))
		}
		uid, gid = int(tid), int(tid)
	}

	return
}

func extractUidGids(ids string) (uid, gid int, gids []int) {
	if ids == "" {
		return
	}

	var tok = strings.Split(ids, ",")
	uid, gid = pwtoid(tok[0])
	if len(tok) < 2 {
		// gid is already set above
		return
	}

	_, gid = pwtoid(tok[1])

	for i := 2; i < len(tok); i++ {
		_, gid := pwtoid(tok[i])
		gids = append(gids, gid)
	}

	if len(gids) >= 10 {
		gids = gids[:10]
		_, _ = fmt.Fprintln(os.Stderr, "too many group ids")
	}

	return
}

func main() {
	var (
		err  error
		args = os.Args[1:]
	)

	//if uid := os.Getuid(); uid != user.AidRoot && uid != user.AidShell {
	//	errorExit(1, nil, "not allowed")
	//}

	var (
		path     bool   // The inherits parent process path env.
		uid, gid = 0, 0 // The default user is root.
	)

	// Handle -h and --help.
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		help13()
		return
	}

	// Handle -p and --path.
	if len(args) > 0 && (args[0] == "--path" || args[0] == "-p") {
		path = true
		args = args[1:]
	}

	// If there are any arguments, the first argument is the uid/gid/supplementary groups.
	if len(args) > 0 {
		var gids []int
		uid, gid, gids = extractUidGids(args[0])
		if len(gids) > 0 {
			if err = syscall.Setgroups(gids); err != nil {
				errorExit(1, err, "setgroups failed")
			}
		}
		args = args[1:]
	}

	if err = syscall.Setgid(gid); err != nil {
		errorExit(1, err, "setgid failed")
	}

	if err = syscall.Setuid(uid); err != nil {
		errorExit(1, err, "setuid failed")
	}

	// Reset parts of the environment.
	if !path {
		_ = os.Setenv("PATH", defPath())
	}
	_ = os.Unsetenv("IFS")
	if pw := user.Getpwuid(uint32(uid)); pw != nil {
		_ = os.Setenv("LOGNAME", pw.Name)
		_ = os.Setenv("USER", pw.Name)
	} else {
		_ = os.Unsetenv("LOGNAME")
		_ = os.Unsetenv("USER")
	}

	// Set up the arguments for exec.
	var execArgs = make([]string, 0, len(args)+1)
	for _, arg := range args {
		execArgs = append(execArgs, arg)
	}

	// Default to the standard shell.
	if len(execArgs) == 0 {
		execArgs = append(execArgs, "/system/bin/sh")
	}

	if err = syscall.Exec(execArgs[0], execArgs, os.Environ()); err != nil {
		errorExit(1, err, fmt.Sprintf("failed to exec %s", execArgs[0]))
	}
}
