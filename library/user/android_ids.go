/**
 * @Author: zzy
 * @Email: zhangzhongyuan@didiglobal.com
 * @Description:
 * @File: android_ids.go
 * @Package: user
 * @Version: 1.0.0
 * @Date: 2022/10/17 16:45
 */

package user

type AndroidIDInfo struct {
	Name string
	Aid  uint32
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
	{"adb", AidAdb},
	{"install", AidInstall},
	{"media", AidMedia},
	{"dhcp", AidDhcp},
	{"sdcard_rw", AidSdcardRW},
	{"vpn", AidVpn},
	{"keystore", AidKeystore},
	{"usb", AidUsb},
	{"drm", AidDrm},
	{"mdnsr", AidMDnsR},
	{"gps", AidGPS},
	// AidUnused1
	{"media_rw", AidMediaRW},
	{"mtp", AidMTP},
	// AidUnused2
	{"drmrpc", AidDrmRpc},
	{"nfc", AidNfc},
	{"sdcard_r", AidSdcardR},
	{"clat", AidClat},
	{"loop_radio", AidLoopRadio},
	{"mediadrm", AidMediaDrm},
	{"package_info", AidPackageInfo},
	{"sdcard_pics", AidSdcardPics},
	{"sdcard_av", AidSdcardAV},
	{"sdcard_all", AidSdcardAll},
	{"logd", AidLogd},
	{"shared_relro", AidSharedRELRO},

	{"dbus", AidDbus},
	{"tlsdate", AidTlsdate},
	{"mediaex", AidMediaEx},
	{"audioserver", AidAudioServer},
	{"metrics_coll", AidMetricsColl},
	{"metricsd", AidMetricSD},
	{"webserv", AidWebServ},
	{"debuggerd", AidDebugGerd},
	{"mediacodec", AidMediaCodec},
	{"cameraserver", AidCameraServer},
	{"firewall", AidFirewall},
	{"trunks", AidTrunks},
	{"nvram", AidNVRAM},
	{"dns", AidDns},
	{"dns_tether", AidDnsTether},

	{"shell", AidShell},
	{"cache", AidCache},
	{"diag", AidDiag},

	{"net_bt_admin", AidNetBtAdmin},
	{"net_bt", AidNetBt},
	{"inet", AidInet},
	{"net_raw", AidNetRaw},
	{"net_admin", AidNetAdmin},
	{"net_bw_stats", AidNetBwStats},
	{"net_bw_acct", AidNetBwAcct},
	{"net_bt_stack", AidNetBtStack},
	{"readproc", AidReadProc},
	{"wakelock", AidWakelock},

	{"everybody", AidEverybody},
	{"misc", AidMisc},
	{"nobody", AidNobody},
}

func AndroidIDCount() int {
	return len(AndroidIDs)
}
