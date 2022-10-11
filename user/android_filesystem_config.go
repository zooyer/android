package pwd

import (
	"fmt"
	"os"
	"strings"
)

/* This is the master Users and Groups config for the platform.
** DO NOT EVER RENUMBER.
 */
const (
	AidRoot      = 0    // traditional unix root user
	AidSystem    = 1000 // system server
	AidRadio     = 1001 // telephony subsystem, RIL
	AidBluetooth = 1002 // bluetooth subsystem
	AidGraphics  = 1003 // graphics devices
	AidInput     = 1004 // input devices
	AidAudio     = 1005 // audio devices
	AidCamera    = 1006 // camera devices
	AidLog       = 1007 // log devices
	AidCompass   = 1008 // compass device
	AidMount     = 1009 // mountd socket
	AidWifi      = 1010 // wifi subsystem
	AidAdb       = 1011 // android debug bridge (adbd)
	AidInstall   = 1012 // group for installing packages
	AidMedia     = 1013 // mediaserver process
	AidDhcp      = 1014 // dhcp client
	AidSdcardRW  = 1015 // external storage write access
	AidVpn       = 1016 // vpn system
	AidKeystore  = 1017 // keystore subsystem
	AidUsb       = 1018 // USB devices
	AidDrm       = 1019 // DRM server
	AidMDnsR     = 1020 // MulticastDNSResponder (service discovery)
	AidGPS       = 1021 // GPS daemon
	AidUnused1   = 1022 // deprecated, DO NOT USE
	AidMediaRW   = 1023 // internal media storage write access
	AidMTP       = 1024 // MTP USB driver access
	AidUnused2   = 1025 // deprecated, DO NOT USE
	AidDrmRpc    = 1026 // group for drm rpc
	AidNfc       = 1027 // nfc subsystem
	AidSdcardR   = 1028 // external storage read access

	AidShell = 2000 // adb and debug shell user
	AidCache = 2001 // cache access
	AidDiag  = 2002 // access to diagnostic resources

	/* The 3000 series are intended for use as supplemental group id's only.
	 * They indicate special Android capabilities that the kernel is aware of. */

	AidNetBtAdmin = 3001 // bluetooth: create any socket
	AidNetBt      = 3002 // bluetooth: create sco, rfcomm or l2cap sockets
	AidInet       = 3003 // can create AF_INET and AF_INET6 sockets
	AidNetRaw     = 3004 // can create raw INET sockets
	AidNetAdmin   = 3005 // can configure interfaces and routing tables.
	AidNetBwStats = 3006 // read bandwidth statistics
	AidNetBwAcct  = 3007 // change bandwidth statistics accounting
	AidNetBtStack = 3008 // bluetooth: access config files

	AidMisc   = 9998 // access to misc storage
	AidNobody = 9999

	AidApp = 10000 // first app user

	AidIsolatedStart = 99000 // start of uids for fully isolated sandboxed processes
	AidIsolatedEnd   = 99999 // end of uids for fully isolated sandboxed processes

	AidUser = 100000 // offset for uid ranges for each user

	AidSharedGidStart = 50000 // start of gids for apps in each user to share
	AidSharedGidEnd   = 59999 // start of gids for apps in each user to share
)

type AndroidIDInfo struct {
	Name string
	Aid  uint
}

var AndroidIDs = []AndroidIDInfo{
	{"root", AidRoot},
	{"system", AidSystem},
	{"radio", AidRadio},
	{"bluetooth", AidBluetooth},
	{"graphics", AidGraphics},
	{"input", AidInput},
	{"audio", AidAudio},
	{"camera", AidCamera},
	{"log", AidLog},
	{"compass", AidCompass},
	{"mount", AidMount},
	{"wifi", AidWifi},
	{"dhcp", AidDhcp},
	{"adb", AidAdb},
	{"install", AidInstall},
	{"media", AidMedia},
	{"drm", AidDrm},
	{"mdnsr", AidMDnsR},
	{"nfc", AidNfc},
	{"drmrpc", AidDrmRpc},
	{"shell", AidShell},
	{"cache", AidCache},
	{"diag", AidDiag},
	{"net_bt_admin", AidNetBtAdmin},
	{"net_bt", AidNetBt},
	{"net_bt_stack", AidNetBtStack},
	{"sdcard_r", AidSdcardR},
	{"sdcard_rw", AidSdcardRW},
	{"media_rw", AidMediaRW},
	{"vpn", AidVpn},
	{"keystore", AidKeystore},
	{"usb", AidUsb},
	{"mtp", AidMTP},
	{"gps", AidGPS},
	{"inet", AidInet},
	{"net_raw", AidNetRaw},
	{"net_admin", AidNetAdmin},
	{"net_bw_stats", AidNetBwStats},
	{"net_bw_acct", AidNetBwAcct},
	{"misc", AidMisc},
	{"nobody", AidNobody},
}

func AndroidIDCount() int {
	return len(AndroidIDs)
}

type FSPathConfig struct {
	Mode   uint
	UID    uint
	GID    uint
	Prefix string
}

/* Rules for directories.
** These rules are applied based on "first match", so they
** should start with the most specific path and work their
** way up to the root.
 */

var AndroidDirs = []FSPathConfig{
	{00770, AidSystem, AidCache, "cache"},
	{00771, AidSystem, AidSystem, "data/app"},
	{00771, AidSystem, AidSystem, "data/app-private"},
	{00771, AidSystem, AidSystem, "data/dalvik-cache"},
	{00771, AidSystem, AidSystem, "data/data"},
	{00771, AidShell, AidShell, "data/local/tmp"},
	{00771, AidShell, AidShell, "data/local"},
	{01771, AidSystem, AidMisc, "data/misc"},
	{00770, AidDhcp, AidDhcp, "data/misc/dhcp"},
	{00775, AidMediaRW, AidMediaRW, "data/media"},
	{00775, AidMediaRW, AidMediaRW, "data/media/Music"},
	{00771, AidSystem, AidSystem, "data"},
	{00750, AidRoot, AidShell, "sbin"},
	{00755, AidRoot, AidShell, "system/bin"},
	{00755, AidRoot, AidShell, "system/vendor"},
	{00755, AidRoot, AidShell, "system/xbin"},
	{00755, AidRoot, AidRoot, "system/etc/ppp"},
	{00777, AidRoot, AidRoot, "sdcard"},
	{00755, AidRoot, AidRoot, ""},
}

/* Rules for files.
** These rules are applied based on "first match", so they
** should start with the most specific path and work their
** way up to the root. Prefixes ending in * denotes wildcard
** and will allow partial matches.
 */

var AndroidFiles = []FSPathConfig{
	{00440, AidRoot, AidShell, "system/etc/init.goldfish.rc"},
	{00550, AidRoot, AidShell, "system/etc/init.goldfish.sh"},
	{00440, AidRoot, AidShell, "system/etc/init.trout.rc"},
	{00550, AidRoot, AidShell, "system/etc/init.ril"},
	{00550, AidRoot, AidShell, "system/etc/init.testmenu"},
	{00550, AidDhcp, AidShell, "system/etc/dhcpcd/dhcpcd-run-hooks"},
	{00440, AidBluetooth, AidBluetooth, "system/etc/dbus.conf"},
	{00444, AidRadio, AidAudio, "system/etc/AudioPara4.csv"},
	{00555, AidRoot, AidRoot, "system/etc/ppp/*"},
	{00555, AidRoot, AidRoot, "system/etc/rc.*"},
	{00644, AidSystem, AidSystem, "data/app/*"},
	{00644, AidMediaRW, AidMediaRW, "data/media/*"},
	{00644, AidSystem, AidSystem, "data/app-private/*"},
	{00644, AidApp, AidApp, "data/data/*"},
	/* the following two files are INTENTIONALLY set-gid and not set-uid.
	 * Do not change. */
	{02755, AidRoot, AidNetRaw, "system/bin/ping"},
	{02750, AidRoot, AidInet, "system/bin/netcfg"},
	/* the following five files are INTENTIONALLY set-uid, but they
	 * are NOT included on user builds. */
	{06755, AidRoot, AidRoot, "system/xbin/su"},
	{06755, AidRoot, AidRoot, "system/xbin/librank"},
	{06755, AidRoot, AidRoot, "system/xbin/procrank"},
	{06755, AidRoot, AidRoot, "system/xbin/procmem"},
	{06755, AidRoot, AidRoot, "system/xbin/tcpdump"},
	{04770, AidRoot, AidRadio, "system/bin/pppd-ril"},
	/* the following file is INTENTIONALLY set-uid, and IS included
	 * in user builds. */
	{06750, AidRoot, AidShell, "system/bin/run-as"},
	{00755, AidRoot, AidShell, "system/bin/*"},
	{00755, AidRoot, AidRoot, "system/lib/valgrind/*"},
	{00755, AidRoot, AidShell, "system/xbin/*"},
	{00755, AidRoot, AidShell, "system/vendor/bin/*"},
	{00750, AidRoot, AidShell, "sbin/*"},
	{00755, AidRoot, AidRoot, "bin/*"},
	{00750, AidRoot, AidShell, "init*"},
	{00750, AidRoot, AidShell, "charger*"},
	{00750, AidRoot, AidShell, "sbin/fs_mgr"},
	{00640, AidRoot, AidShell, "fstab.*"},
	{00644, AidRoot, AidRoot, ""},
}

func FSConfig(path string, dir bool) (uid, gid, mode uint) {
	var pc = AndroidFiles
	if dir {
		pc = AndroidDirs
	}

	var p FSPathConfig
	var pathLen = len(path)
	for _, p = range pc {
		if p.Prefix == "" {
			continue
		}

		var prefixLen = len(p.Prefix)
		if dir {
			if pathLen < prefixLen {
				continue
			}
			if strings.HasPrefix(path, p.Prefix) {
				break
			}
			continue
		}
		// If name ends in * then allow partial matches.
		if p.Prefix[prefixLen-1] == '*' {
			if path == p.Prefix[:prefixLen-1] {
				break
			}
		} else if pathLen == prefixLen {
			if path == p.Prefix {
				break
			}
		}
	}

	uid = p.UID
	gid = p.GID
	mode = p.Mode

	if false {
		fmt.Fprintf(os.Stderr, "< '%s' '%s' %d %d %o >\n", path, p.Prefix, uid, gid, mode)
	}

	return
}
