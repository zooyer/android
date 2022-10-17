package user

// generator script: https://android.googlesource.com/platform/build/+/refs/heads/android13-s3-release/tools/fs_config/fs_config_generator.py

/* This is the master Users and Groups config for the platform.
 * DO NOT EVER RENUMBER.
 */
const (
	AidRoot = 0 // traditional unix root user

	/* The following are for LTP and should only be used for testing */

	AidDaemon = 1 // traditional unix daemon owner
	AIdBin    = 2 // traditional unix binaries owner

	AidSystem                = 1000 // system server
	AidRadio                 = 1001 // telephony subsystem, RIL
	AidBluetooth             = 1002 // bluetooth subsystem
	AidGraphics              = 1003 // graphics devices
	AidInput                 = 1004 // input devices
	AidAudio                 = 1005 // audio devices
	AidCamera                = 1006 // camera devices
	AidLog                   = 1007 // log devices
	AidCompass               = 1008 // compass device
	AidMount                 = 1009 // mountd socket
	AidWifi                  = 1010 // wifi subsystem
	AidAdb                   = 1011 // android debug bridge (adbd)
	AidInstall               = 1012 // group for installing packages
	AidMedia                 = 1013 // mediaserver process
	AidDhcp                  = 1014 // dhcp client
	AidSdcardRW              = 1015 // external storage write access
	AidVpn                   = 1016 // vpn system
	AidKeystore              = 1017 // keystore subsystem
	AidUsb                   = 1018 // USB devices
	AidDrm                   = 1019 // DRM server
	AidMDnsR                 = 1020 // MulticastDNSResponder (service discovery)
	AidGPS                   = 1021 // GPS daemon
	AidUnused1               = 1022 // deprecated, DO NOT USE
	AidMediaRW               = 1023 // internal media storage write access
	AidMTP                   = 1024 // MTP USB driver access
	AidUnused2               = 1025 // deprecated, DO NOT USE
	AidDrmRpc                = 1026 // group for drm rpc
	AidNfc                   = 1027 // nfc subsystem
	AidSdcardR               = 1028 // external storage read access
	AidClat                  = 1029 // clat part of nat464
	AidLoopRadio             = 1030 // loop radio devices
	AidMediaDrm              = 1031 // MediaDrm plugins
	AidPackageInfo           = 1032 // access to installed package details
	AidSdcardPics            = 1033 // external storage photos access
	AidSdcardAV              = 1034 // external storage audio/video access
	AidSdcardAll             = 1035 // access all users external storage
	AidLogd                  = 1036 // log daemon
	AidSharedRELRO           = 1037 // creator of shared GNU RELRO files
	AidDbus                  = 1038 // dbus-daemon IPC broker process
	AidTlsdate               = 1039 // tlsdate unprivileged user
	AidMediaEx               = 1040 // mediaextractor process
	AidAudioServer           = 1041 // audioserver process
	AidMetricsColl           = 1042 // metrics_collector process
	AidMetricSD              = 1043 // metricsd process
	AidWebServ               = 1044 // webservd process
	AidDebugGerd             = 1045 // debuggerd unprivileged user
	AidMediaCodec            = 1046 // mediacodec process
	AidCameraServer          = 1047 // cameraserver process
	AidFirewall              = 1048 // firewalld process
	AidTrunks                = 1049 // trunksd process (TPM daemon)
	AidNVRAM                 = 1050 // Access-controlled NVRAM
	AidDns                   = 1051 // DNS resolution daemon (system: netd)
	AidDnsTether             = 1052 // DNS resolution daemon (tether: dnsmasq)
	AidWebviewZygote         = 1053 // WebView zygote process
	AidVehicleNetwork        = 1054 // Vehicle network service
	AidMediaAudio            = 1055 // GID for audio files on internal media storage
	AidMediaVideo            = 1056 // GID for video files on internal media storage
	AidMediaImage            = 1057 // GID for image files on internal media storage
	AidTombstoned            = 1058 // tombstoned user
	AidMediaOBB              = 1059 // GID for OBB files on internal media storage
	AidESE                   = 1060 // embedded secure element (eSE) subsystem
	AidOTAUpdate             = 1061 // resource tracking UID for OTA updates
	AidAutomotiveEvs         = 1062 // Automotive rear and surround view system
	AidLoWPAN                = 1063 // LoWPAN subsystem
	AidHsm                   = 1064 // hardware security module subsystem
	AidReservedDisk          = 1065 // GID that has access to reserved disk space
	AidStatsd                = 1066 // statsd daemon
	AidIncidentd             = 1067 // incidentd daemon
	AidSecureElement         = 1068 // secure element subsystem
	AidLMKD                  = 1069 // low memory killer daemon
	AidLLKD                  = 1070 // live lock daemon
	AidIORAPD                = 1071 // input/output readahead and pin daemon
	AidGPUService            = 1072 // GPU service daemon
	AidNetworkStack          = 1073 // network stack service
	AidGSID                  = 1074 // GSI service daemon
	AidFSVerityCert          = 1075 // fs-verity key ownership in keystore
	AidCredStore             = 1076 // identity credential manager service
	AidExternalStorage       = 1077 // Full external storage access including USB OTG volumes
	AidExtDataRW             = 1078 // GID for app-private data directories on external storage
	AidExtObbRW              = 1079 // GID for OBB directories on external storage
	AidContextHub            = 1080 // GID for access to the Context Hub
	AidVirtualizationService = 1081 // VirtualizationService daemon
	AidARTD                  = 1082 // ART Service daemon
	AidUWB                   = 1083 // UWB subsystem
	AidThreadNetwork         = 1084 // Thread Network subsystem
	AidDiced                 = 1085 // Android's DICE daemon
	AidDmesgd                = 1086 // dmesg parsing daemon for kernel report collection
	AidJcWeaver              = 1087 // Javacard Weaver HAL - to manage omapi ARA rules
	AidJcStrongbox           = 1088 // Javacard Strongbox HAL - to manage omapi ARA rules
	AidJcIdentityCred        = 1089 // Javacard Identity Cred HAL - to manage omapi ARA rules
	AidSDKSandbox            = 1090 // SDK sandbox virtual UID
	AidSecurityLogWriter     = 1091 // write to security log

	/* Changes to this file must be made in AOSP, *not* in internal branches. */

	AidShell = 2000 // adb and debug shell user
	AidCache = 2001 // cache access
	AidDiag  = 2002 // access to diagnostic resources

	/* The range 2900-2999 is reserved for the vendor partition */
	/* Note that the two 'OEM' ranges pre-dated the vendor partition, so they take the legacy 'OEM'
	 * name. Additionally, they pre-dated passwd/group files, so there are users and groups named oem_#
	 * created automatically for all values in these ranges.  If there is a user/group in a passwd/group
	 * file corresponding to this range, both the oem_# and user/group names will resolve to the same
	 * value. */

	AidOemReservedStart = 2900
	AidOemReservedEnd   = 2999

	/* The 3000 series are intended for use as supplemental group id's only.
	 * They indicate special Android capabilities that the kernel is aware of. */

	AidNetBtAdmin  = 3001 // bluetooth: create any socket
	AidNetBt       = 3002 // bluetooth: create sco, rfcomm or l2cap sockets
	AidInet        = 3003 // can create AF_INET and AF_INET6 sockets
	AidNetRaw      = 3004 // can create raw INET sockets
	AidNetAdmin    = 3005 // can configure interfaces and routing tables.
	AidNetBwStats  = 3006 // read bandwidth statistics
	AidNetBwAcct   = 3007 // change bandwidth statistics accounting
	AidNetBtStack  = 3008 // bluetooth: access config files
	AidReadProc    = 3009 // Allow /proc read access
	AidWakelock    = 3010 // Allow system wakelock read/write access
	AidUhid        = 3011 // Allow read/write to /dev/uhid node
	AidReadTracefs = 3012 // Allow tracefs read

	/* The range 5000-5999 is also reserved for vendor partition. */

	AidOemReserved2Start = 5000
	AidOemReserved2End   = 5999

	/* The range 6000-6499 is reserved for the system partition. */

	AidSystemReservedStart = 6000
	AidSystemReservedEnd   = 6499

	/* The range 6500-6999 is reserved for the odm partition. */

	AidOdmReservedStart = 6500
	AidOdmReservedEnd   = 6999

	/* The range 7000-7499 is reserved for the product partition. */

	AidProductReservedStart = 7000
	AidProductReservedEnd   = 7499

	/* The range 7500-7999 is reserved for the system_ext partition. */

	AidSystemExtReservedStart = 7500
	AidSystemExtReservedEnd   = 7999
	AidEverybody              = 9997 // shared between all apps in the same profile
	AidMisc                   = 9998 // access to misc storage
	AidNobody                 = 9999

	AidApp      = 10000 // TODO: switch users over to AidAppStart
	AidAppStart = 10000 // first app user
	AidAppEnd   = 19999 // last app user

	AidCacheGidStart = 20000 // start of gids for apps to mark cached data
	AidCacheGidEnd   = 29999 // end of gids for apps to mark cached data

	AidExtGidStart = 30000 // start of gids for apps to mark external data
	AidExtGidEnd   = 39999 // end of gids for apps to mark external data

	AidExtCacheGidStart = 40000 // start of gids for apps to mark external cached data
	AidExtCacheGidEnd   = 49999 // end of gids for apps to mark external cached data

	AidSharedGidStart = 50000 // start of gids for apps in each user to share
	AidSharedGidEnd   = 59999 // end of gids for apps in each user to share

	/*
	 * This is a magic number in the kernel and not something that was picked
	 * arbitrarily. This value is returned whenever a uid that has no mapping in the
	 * user namespace is returned to userspace:
	 * https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/include/linux/highuid.h?h=v4.4#n40
	 */

	AidOverFlowUid = 65534 // unmapped user in the user namespace

	/* use the ranges below to determine whether a process is sdk sandbox */

	AidSdkSandboxProcessStart = 20000 // start of uids allocated to sdk sandbox processes
	AidSdkSandboxProcessEnd   = 29999 // end of uids allocated to sdk sandbox processes

	/* use the ranges below to determine whether a process is isolated */

	AidIsolatedStart = 90000 // start of uids for fully isolated sandboxed processes
	AidIsolatedEnd   = 99999 // end of uids for fully isolated sandboxed processes

	AidUser       = 100000 // TODO: switch users over to AidUserOffset
	AidUserOffset = 100000 // offset for uid ranges for each user
)
