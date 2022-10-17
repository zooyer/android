/**
 * @Author: zzy
 * @Email: zhangzhongyuan@didiglobal.com
 * @Description:
 * @File: fs_config.go
 * @Package: user
 * @Version: 1.0.0
 * @Date: 2022/10/17 14:13
 */

package user

import (
	"strings"
)

type FSPathConfig struct {
	Mode         uint
	UID          uint
	GID          uint
	Capabilities uint64
	Prefix       string
}

// AndroidDirs
// Rules for directories.
// These rules are applied based on "first match", so they
// should start with the most specific path and work their
// way up to the root.
var AndroidDirs = []FSPathConfig{
	// clang-format off
	{00770, AidSystem, AidCache, 0, "cache"},
	{00555, AidRoot, AidRoot, 0, "config"},
	{00771, AidSystem, AidSystem, 0, "data/app"},
	{00771, AidSystem, AidSystem, 0, "data/app-private"},
	{00771, AidSystem, AidSystem, 0, "data/app-ephemeral"},
	{00771, AidRoot, AidRoot, 0, "data/dalvik-cache"},
	{00771, AidSystem, AidSystem, 0, "data/data"},
	{00771, AidShell, AidShell, 0, "data/local/tmp"},
	{00771, AidShell, AidShell, 0, "data/local"},
	{00770, AidDhcp, AidDhcp, 0, "data/misc/dhcp"},
	{00771, AidSharedRELRO, AidSharedRELRO, 0, "data/misc/shared_relro"},
	{01771, AidSystem, AidMisc, 0, "data/misc"},
	{00775, AidMediaRW, AidMediaRW, 0, "data/media/Music"},
	{00775, AidMediaRW, AidMediaRW, 0, "data/media"},
	{00750, AidRoot, AidShell, 0, "data/nativetest"},
	{00750, AidRoot, AidShell, 0, "data/nativetest64"},
	{00750, AidRoot, AidShell, 0, "data/benchmarktest"},
	{00750, AidRoot, AidShell, 0, "data/benchmarktest64"},
	{00775, AidRoot, AidRoot, 0, "data/preloads"},
	{00771, AidSystem, AidSystem, 0, "data"},
	{00755, AidRoot, AidSystem, 0, "mnt"},
	{00751, AidRoot, AidShell, 0, "product/bin"},
	{00751, AidRoot, AidShell, 0, "product/apex/*/bin"},
	{00777, AidRoot, AidRoot, 0, "sdcard"},
	{00751, AidRoot, AidSdcardR, 0, "storage"},
	{00751, AidRoot, AidShell, 0, "system/bin"},
	{00755, AidRoot, AidRoot, 0, "system/etc/ppp"},
	{00755, AidRoot, AidShell, 0, "system/vendor"},
	{00750, AidRoot, AidShell, 0, "system/xbin"},
	{00751, AidRoot, AidShell, 0, "system/apex/*/bin"},
	{00751, AidRoot, AidShell, 0, "system_ext/bin"},
	{00751, AidRoot, AidShell, 0, "system_ext/apex/*/bin"},
	{00751, AidRoot, AidShell, 0, "vendor/bin"},
	{00751, AidRoot, AidShell, 0, "vendor/apex/*/bin"},
	{00755, AidRoot, AidShell, 0, "vendor"},
	{00755, AidRoot, AidRoot, 0, ""},
	// clang-format on
}

// Rules for files.
// These rules are applied based on "first match", so they
// should start with the most specific path and work their
// way up to the root. Prefixes ending in * denotes wildcard
// and will allow partial matches.
var (
	sysConfDir  = "/system/etc/fs_config_dirs"
	sysConfFile = "/system/etc/fs_config_files"
)

// No restrictions are placed on the vendor and oem file-system config files,
// although the developer is advised to restrict the scope to the /vendor or
// oem/ file-system since the intent is to provide support for customized
// portions of a separate vendor.img or oem.img.  Has to remain open so that
// customization can also land on /system/vendor, /system/oem, /system/odm,
// /system/product or /system/system_ext.
//
// We expect build-time checking or filtering when constructing the associated
// fs_config_* files (see build/tools/fs_config/fs_config_generate.c)
var (
	venConfDir        = "/vendor/etc/fs_config_dirs"
	venConfFile       = "/vendor/etc/fs_config_files"
	oemConfDir        = "/oem/etc/fs_config_dirs"
	oemConfFile       = "/oem/etc/fs_config_files"
	odmConfDir        = "/odm/etc/fs_config_dirs"
	odmConfFile       = "/odm/etc/fs_config_files"
	productConfDir    = "/product/etc/fs_config_dirs"
	productConfFile   = "/product/etc/fs_config_files"
	systemExtConfDir  = "/system_ext/etc/fs_config_dirs"
	systemExtConfFile = "/system_ext/etc/fs_config_files"
)

var conf = [][2]string{
	{sysConfFile, sysConfDir}, {venConfFile, venConfDir},
	{oemConfFile, oemConfDir}, {odmConfFile, odmConfDir},
	{productConfFile, productConfDir}, {systemExtConfFile, systemExtConfDir},
}

// AndroidFiles
// Do not use android_files to grant Linux capabilities.  Use ambient capabilities in their
// associated init.rc file instead.  See https://source.android.com/devices/tech/config/ambient.
// Do not place any new vendor/, data/vendor/, etc entries in android_files.
// Vendor entries should be done via a vendor or device specific config.fs.
// See https://source.android.com/devices/tech/config/filesystem#using-file-system-capabilities
var AndroidFiles = []FSPathConfig{
	// clang-format off
	{00644, AidSystem, AidSystem, 0, "data/app/*"},
	{00644, AidSystem, AidSystem, 0, "data/app-ephemeral/*"},
	{00644, AidSystem, AidSystem, 0, "data/app-private/*"},
	{00644, AidApp, AidApp, 0, "data/data/*"},
	{00644, AidMediaRW, AidMediaRW, 0, "data/media/*"},
	{00640, AidRoot, AidShell, 0, "data/nativetest/tests.txt"},
	{00640, AidRoot, AidShell, 0, "data/nativetest64/tests.txt"},
	{00750, AidRoot, AidShell, 0, "data/nativetest/*"},
	{00750, AidRoot, AidShell, 0, "data/nativetest64/*"},
	{00750, AidRoot, AidShell, 0, "data/benchmarktest/*"},
	{00750, AidRoot, AidShell, 0, "data/benchmarktest64/*"},
	{00600, AidRoot, AidRoot, 0, "default.prop"}, // legacy
	{00600, AidRoot, AidRoot, 0, "system/etc/prop.default"},
	{00600, AidRoot, AidRoot, 0, "odm/build.prop"},   // legacy; only for P release
	{00600, AidRoot, AidRoot, 0, "odm/default.prop"}, // legacy; only for P release
	{00600, AidRoot, AidRoot, 0, "odm/etc/build.prop"},
	{00444, AidRoot, AidRoot, 0, odmConfDir[1:]},
	{00444, AidRoot, AidRoot, 0, odmConfFile[1:]},
	{00444, AidRoot, AidRoot, 0, oemConfDir[1:]},
	{00444, AidRoot, AidRoot, 0, oemConfFile[1:]},
	{00600, AidRoot, AidRoot, 0, "product/build.prop"},
	{00444, AidRoot, AidRoot, 0, productConfDir[1:]},
	{00444, AidRoot, AidRoot, 0, productConfFile[1:]},
	{00600, AidRoot, AidRoot, 0, "system_ext/build.prop"},
	{00444, AidRoot, AidRoot, 0, systemExtConfDir[1:]},
	{00444, AidRoot, AidRoot, 0, systemExtConfFile[1:]},
	{00755, AidRoot, AidShell, 0, "system/bin/crash_dump32"},
	{00755, AidRoot, AidShell, 0, "system/bin/crash_dump64"},
	{00755, AidRoot, AidShell, 0, "system/bin/debuggerd"},
	{00550, AidLogd, AidLogd, 0, "system/bin/logd"},
	{00700, AidRoot, AidRoot, 0, "system/bin/secilc"},
	{00750, AidRoot, AidRoot, 0, "system/bin/uncrypt"},
	{00600, AidRoot, AidRoot, 0, "system/build.prop"},
	{00444, AidRoot, AidRoot, 0, sysConfDir[1:]},
	{00444, AidRoot, AidRoot, 0, sysConfFile[1:]},
	{00440, AidRoot, AidShell, 0, "system/etc/init.goldfish.rc"},
	{00550, AidRoot, AidShell, 0, "system/etc/init.goldfish.sh"},
	{00550, AidRoot, AidShell, 0, "system/etc/init.ril"},
	{00555, AidRoot, AidRoot, 0, "system/etc/ppp/*"},
	{00555, AidRoot, AidRoot, 0, "system/etc/rc.*"},
	{00750, AidRoot, AidRoot, 0, "vendor/bin/install-recovery.sh"},
	{00600, AidRoot, AidRoot, 0, "vendor/build.prop"},
	{00600, AidRoot, AidRoot, 0, "vendor/default.prop"},
	{00440, AidRoot, AidRoot, 0, "vendor/etc/recovery.img"},
	{00444, AidRoot, AidRoot, 0, venConfDir[1:]},
	{00444, AidRoot, AidRoot, 0, venConfFile[1:]},
	// the following two files are INTENTIONALLY set-uid, but they
	// are NOT included on user builds.
	{06755, AidRoot, AidRoot, 0, "system/xbin/procmem"},
	{04750, AidRoot, AidShell, 0, "system/xbin/su"},
	// the following files have enhanced capabilities and ARE included
	// in user builds.
	//{00700, AidSystem, AidShell, CAP_MASK_LONG(CAP_BLOCK_SUSPEND),
	//	"system/bin/inputflinger"},
	//{00750, AidRoot, AidShell, CAP_MASK_LONG(CAP_SETUID) |
	//	CAP_MASK_LONG(CAP_SETGID),
	//	"system/bin/run-as"},
	//{00750, AidRoot, AidShell, CAP_MASK_LONG(CAP_SETUID) |
	//	CAP_MASK_LONG(CAP_SETGID),
	//	"system/bin/simpleperf_app_runner"},
	{00755, AidRoot, AidRoot, 0, "first_stage_ramdisk/system/bin/e2fsck"},

	//	#ifdef __LP64__
	//{ 00755, AidRoot,      AidRoot,      0, "first_stage_ramdisk/system/bin/linker64" },
	//	#else
	//{ 00755, AidRoot,      AidRoot,      0, "first_stage_ramdisk/system/bin/linker" },
	//	#endif

	{00755, AidRoot, AidRoot, 0, "first_stage_ramdisk/system/bin/resize2fs"},
	{00755, AidRoot, AidRoot, 0, "first_stage_ramdisk/system/bin/snapuserd"},
	{00755, AidRoot, AidRoot, 0, "first_stage_ramdisk/system/bin/tune2fs"},
	{00755, AidRoot, AidRoot, 0, "first_stage_ramdisk/system/bin/fsck.f2fs"},
	// generic defaults
	{00755, AidRoot, AidRoot, 0, "bin/*"},
	{00640, AidRoot, AidShell, 0, "fstab.*"},
	{00750, AidRoot, AidShell, 0, "init*"},
	{00755, AidRoot, AidShell, 0, "odm/bin/*"},
	{00755, AidRoot, AidShell, 0, "product/bin/*"},
	{00755, AidRoot, AidShell, 0, "product/apex/*bin/*"},
	{00755, AidRoot, AidShell, 0, "system/bin/*"},
	{00755, AidRoot, AidShell, 0, "system/xbin/*"},
	{00755, AidRoot, AidShell, 0, "system/apex/*/bin/*"},
	{00755, AidRoot, AidShell, 0, "system_ext/bin/*"},
	{00755, AidRoot, AidShell, 0, "system_ext/apex/*/bin/*"},
	{00755, AidRoot, AidShell, 0, "vendor/bin/*"},
	{00755, AidRoot, AidShell, 0, "vendor/apex/*bin/*"},
	{00755, AidRoot, AidShell, 0, "vendor/xbin/*"},
	{00644, AidRoot, AidRoot, 0, ""},
	// clang-format on
}

func isPartition(path string) bool {
	var partitions = []string{"odm/", "oem/", "product/", "system_ext/", "vendor/"}

	for _, partition := range partitions {
		if strings.HasPrefix(path, partition) {
			return true
		}
	}

	return false
}
