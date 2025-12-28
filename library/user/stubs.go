package user

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"syscall"
	"unicode"
)

type Group struct {
	Name string
	GID  uint32
}

type Passwd struct {
	Name  string // user's login name
	UID   uint32 // numerical user ID
	GID   uint32 // numerical group ID
	Dir   string // initial working directory
	Shell string // program to use as shell
}

var passwdFiles = [][]string{
	{"/etc/passwd", "system_"}, // symlinks to /system/etc/passwd in Android
	{"/vendor/etc/passwd", "vendor_"},
	{"/odm/etc/passwd", "odm_"},
	{"/product/etc/passwd", "product_"},
	{"/system_ext/etc/passwd", "system_ext_"},
}

var groupFiles = [][]string{
	{"/etc/group", "system_"}, // symlinks to /system/etc/group in Android
	{"/vendor/etc/group", "vendor_"},
	{"/odm/etc/group", "odm_"},
	{"/product/etc/group", "product_"},
	{"/system_ext/etc/group", "system_ext_"},
}

func strtoul(str string, base int, bit int) (num uint32, end string, err error) {
	var (
		i int
		c rune
		s = []rune(str)
	)

	var isBreak bool
	for i, c = range s {
		if !unicode.IsNumber(c) {
			isBreak = true
			break
		}
	}

	if isBreak {
		end = string(s[i:])
	} else {
		i++
	}

	u, err := strconv.ParseUint(string(s[:i]), base, bit)
	if err != nil {
		return
	}

	return uint32(u), end, nil
}

func isDigit(str string, index int) bool {
	return unicode.IsDigit([]rune(str)[index])
}

func androidIInfoToPasswd(info AndroidIDInfo) *Passwd {
	var shell = "/bin/sh"
	if _, err := os.Stat(shell); err != nil {
		shell = "/system/bin/sh"
	}

	return &Passwd{
		Name:  info.Name,
		UID:   info.Aid,
		GID:   info.Aid,
		Dir:   "/",
		Shell: shell,
	}
}

func androidIInfoToGroup(info AndroidIDInfo) *Group {
	return &Group{
		Name: info.Name,
		GID:  info.Aid,
	}
}

func findAndroidIDInfoByID(id uint32) *AndroidIDInfo {
	for _, aid := range AndroidIDs {
		if aid.Aid == id {
			return &aid
		}
	}

	return nil
}

func findAndroidIDInfoByName(name string) *AndroidIDInfo {
	for _, aid := range AndroidIDs {
		if aid.Name == name {
			return &aid
		}
	}

	return nil
}

// These are a list of the reserved app ranges, and should never contain anything below
// AID_APP_START.  They exist per user, so a given uid/gid modulo AID_USER_OFFSET will map
// to these ranges.
type idRange struct {
	Start, End uint32
}

var userRanges = []idRange{
	{AidAppStart, AidAppEnd},
	{AidIsolatedStart, AidIsolatedEnd},
}

var groupRanges = []idRange{
	{AidAppStart, AidAppEnd},
	{AidCacheGidStart, AidCacheGidEnd},
	{AidExtGidStart, AidExtGidEnd},
	{AidExtCacheGidStart, AidExtCacheGidEnd},
	{AidSharedGidStart, AidSharedGidEnd},
	{AidIsolatedStart, AidIsolatedEnd},
}

func verifyUserRangesAscending[T []idRange](ranges T) bool {
	if len(ranges) < 2 {
		return false
	}

	if ranges[0].Start > ranges[0].End {
		return false
	}

	for i := 1; i < len(ranges); i++ {
		if ranges[i].Start > ranges[i].End {
			return false
		}
		if ranges[i-1].End > ranges[i].Start {
			return false
		}
	}

	return true
}

func init() {
	if !verifyUserRangesAscending(userRanges) {
		panic("user_ranges must have ascending ranges")
	}
	if !verifyUserRangesAscending(groupRanges) {
		panic("user_ranges must have ascending ranges")
	}
}

// This list comes from PackageManagerService.java, where platform AIDs are added to list of valid
// AIDs for packages via addSharedUserLPw().
var secondaryUserPlatformIDs = []uint32{
	AidSystem, AidRadio, AidLog, AidNfc, AidBluetooth,
	AidShell, AidSecureElement, AidNetworkStack,
}

func platformIDSecondaryUserAllowed(id uint32) bool {
	for _, allowedID := range secondaryUserPlatformIDs {
		if allowedID == id {
			return true
		}
	}

	return false
}

func isValidAppID(id uint32, isGroup bool) bool {
	var appid = id % AidUserOffset

	// AidOverFlowUid is never a valid app id, so we explicitly return false to ensure this.
	// This is true across all users, as there is no reason to ever map this id into any user range.
	if appid == AidOverFlowUid {
		return false
	}

	var ranges = userRanges
	if isGroup {
		ranges = groupRanges
	}

	// If we're checking an appid that resolves below the user range, then it's a platform AID for a
	// secondary user. We only allow a reduced set of these, so we must check that it is allowed.
	if appid < ranges[0].Start && platformIDSecondaryUserAllowed(appid) {
		return true
	}

	// The shared GID range is only valid for the first user.
	if appid >= AidSharedGidStart && appid <= AidSharedGidEnd && appid != id {
		return false
	}

	// Otherwise check that the appid is in one of the reserved ranges.
	for _, r := range ranges {
		if appid >= r.Start && appid <= r.End {
			return true
		}
	}

	return false
}

// This provides an iterator for app_ids within the first user's app id's.
func getNextAppID(currentID uint32, isGroup bool) uint32 {
	var ranges = userRanges
	if isGroup {
		ranges = groupRanges
	}

	// If current_id is below the first of the ranges, then we're uninitialized, and return the first
	// valid id.
	if currentID < ranges[0].Start {
		return ranges[0].Start
	}

	var incrementedID = currentID + 1
	// Check to see if our incremented_id is between two ranges, and if so, return the beginning of
	// the next valid range.
	for i := 1; i < len(ranges); i++ {
		if incrementedID > ranges[i-1].End && incrementedID < ranges[i].Start {
			return ranges[i].Start
		}
	}

	// Check to see if our incremented_id is above final range, and return -1 to indicate that we've
	// completed if so.
	if incrementedID > ranges[len(ranges)-1].End {
		return math.MaxUint32 // -1
	}

	// Otherwise the incremented_id is valid, so return it.
	return incrementedID
}

// Translate a user/group name to the corresponding user/group id.
// all_a1234 -> 0 * AID_USER_OFFSET + AID_SHARED_GID_START + 1234 (group name only)
// u0_a1234_ext_cache -> 0 * AID_USER_OFFSET + AID_EXT_CACHE_GID_START + 1234 (group name only)
// u0_a1234_ext -> 0 * AID_USER_OFFSET + AID_EXT_GID_START + 1234 (group name only)
// u0_a1234_cache -> 0 * AID_USER_OFFSET + AID_CACHE_GID_START + 1234 (group name only)
// u0_a1234 -> 0 * AID_USER_OFFSET + AID_APP_START + 1234
// u2_i1000 -> 2 * AID_USER_OFFSET + AID_ISOLATED_START + 1000
// u1_system -> 1 * AID_USER_OFFSET + android_ids['system']
// returns 0 and sets errno to ENOENT in case of error.
func appIDFromName(name string, isGroup bool) (id uint32, err error) {
	var (
		end         string
		isSharedGid bool
		userid      uint32
	)

	if isGroup && len(name) > 2 && name[:3] == "all" {
		end = name[3:]
		userid = 0
		isSharedGid = true
	} else if len(name) > 1 && name[0] == 'u' && isDigit(name, 1) {
		if userid, end, err = strtoul(name[1:], 10, 32); err != nil {
			return
		}
	} else {
		err = syscall.ENOENT
		return
	}

	if !strings.HasPrefix(end, "_") || end[1:] == "" {
		err = syscall.ENOENT
		return
	}

	var appid uint32
	if end[1] == 'a' && len(end) > 2 && isDigit(end, 2) {
		if isSharedGid {
			// end will point to \0 if the strtoul below succeeds.
			if appid, end, err = strtoul(end[2:], 10, 32); err != nil {
				return
			}
			if appid += AidSharedGidStart; appid > AidSharedGidEnd {
				err = syscall.ENOENT
				return
			}
		} else {
			// end will point to \0 if the strtoul below succeeds.
			if appid, end, err = strtoul(end[2:], 10, 32); err != nil {
				return
			}
			if isGroup {
				if end == "_ext_cache" {
					end = end[10:]
					appid += AidExtCacheGidStart
				} else if end == "_ext" {
					end = end[4:]
					appid += AidExtGidStart
				} else if end == "_cache" {
					end = end[6:]
					appid += AidCacheGidStart
				} else {
					appid += AidAppStart
				}
			} else {
				appid += AidAppStart
			}
		}
	} else if end[1] == 'i' && len(end) > 2 && isDigit(end, 2) {
		// end will point to \0 if the strtoul below succeeds.
		if appid, end, err = strtoul(end[2:], 10, 32); err != nil {
			return
		}
		appid += AidIsolatedStart
	} else if info := findAndroidIDInfoByName(end[1:]); info != nil {
		appid = info.Aid
		end = end[len(info.Name):]
	}

	// Check that the entire string was consumed by one of the 3 cases above.
	if end != "" {
		err = syscall.ENOENT
		return
	}

	// Check that user id won't overflow.
	if userid > 1000 {
		err = syscall.ENOENT
		return
	}

	// Check that app id is within range.
	if appid >= AidUserOffset {
		err = syscall.ENOENT
		return
	}

	return appid + userid*AidUserOffset, nil

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

	return uint32(i), nil

FAIL:
	return 0, syscall.ENOENT
}

func printAppNameFromUid(uid uint32) string {
	var (
		appid  = uid % AidUserOffset
		userid = uid / AidUserOffset
	)

	if appid >= AidIsolatedStart {
		return fmt.Sprintf("u%d_i%d", userid, appid-AidIsolatedStart)
	} else if appid < AidAppStart {
		if info := findAndroidIDInfoByID(appid); info != nil {
			return fmt.Sprintf("u%d_%s", userid, info.Name)
		}
	} else {
		return fmt.Sprintf("u%d_a%d", userid, appid-AidAppStart)
	}

	return ""
}

func printAppNameFromGid(gid uint32) string {
	var (
		appid  = gid % AidUserOffset
		userid = gid / AidUserOffset
	)

	if appid >= AidIsolatedStart {
		return fmt.Sprintf("u%d_i%d", userid, appid-AidIsolatedStart)
	} else if userid == 0 && appid >= AidSharedGidStart && appid <= AidSharedGidEnd {
		return fmt.Sprintf("all_a%d", appid-AidSharedGidStart)
	} else if appid >= AidExtCacheGidStart && appid <= AidExtCacheGidEnd {
		return fmt.Sprintf("u%d_a%d_ext_cache", userid, appid-AidExtCacheGidStart)
	} else if appid >= AidExtGidStart && appid <= AidExtGidEnd {
		return fmt.Sprintf("u%d_a%d_ext", userid, appid-AidExtGidStart)
	} else if appid >= AidCacheGidStart && appid <= AidCacheGidEnd {
		return fmt.Sprintf("u%d_a%d_cache", userid, appid-AidCacheGidStart)
	} else if appid < AidAppStart {
		if info := findAndroidIDInfoByID(appid); info != nil {
			return fmt.Sprintf("u%d_%s", userid, info.Name)
		}
	} else {
		return fmt.Sprintf("u%d_a%d", userid, appid-AidAppStart)
	}

	return ""
}

// TODO implement
func deviceLaunchedBeforeApi29() bool {
	/*
	  // Check if ro.product.first_api_level is set to a value > 0 and < 29, if so, this device was
	  // launched before API 29 (Q). Any other value is considered to be either in development or
	  // launched after.
	  // Cache the value as __system_property_get() is expensive and this may be called often.
	  static bool result = [] {
	    char value[PROP_VALUE_MAX] = { 0 };
	    if (__system_property_get("ro.product.first_api_level", value) == 0) {
	      return false;
	    }
	    int value_int = atoi(value);
	    return value_int != 0 && value_int < 29;
	  }();
	  return result;
	*/

	return false
}

// oem_XXXX -> uid
//  Supported ranges:
//   AidOemReservedStart to AidOemReservedEnd (2900-2999)
//   AidOemReserved2Start to AidOemReserved2End (5000-5999)
// Check OEM id is within range.
func isOemID(id uint32) bool {
	if deviceLaunchedBeforeApi29() && id >= AidOemReservedStart && id < AidEverybody && findAndroidIDInfoByID(id) == nil {
		return true
	}

	return id >= AidOemReservedStart && id <= AidOemReservedEnd || id >= AidOemReserved2Start && id <= AidOemReserved2End
}

// Translate an OEM name to the corresponding user/group id.
func oemIDFromName(name string) (id uint32) {
	if _, err := fmt.Sscanf(name, "oem_%d", &id); err != nil {
		return
	}

	if !isOemID(id) {
		return 0
	}

	return
}

func oemIDToPasswd(uid uint32) *Passwd {
	for _, file := range passwdFiles {
		_ = file
		//findByID(uid)
		//return passwd
	}

	if !isOemID(uid) {
		return nil
	}

	var shell = "/bin/sh"
	if _, err := os.Stat(shell); err != nil {
		shell = "/system/bin/sh"
	}

	var pw = Passwd{
		Name:  fmt.Sprintf("oem_%d", uid),
		UID:   uid,
		GID:   uid,
		Dir:   "/",
		Shell: shell,
	}

	return &pw
}

func oemIDToGroup(gid uint32) *Group {
	for _, file := range groupFiles {
		// TODO implement
		_ = file
	}

	if !isOemID(gid) {
		return nil
	}

	var group = Group{
		Name: fmt.Sprintf("oem_%d", gid),
		GID:  gid,
	}

	return &group
}

// Translate a uid into the corresponding name.
// 0 to AID_APP_START-1                    -> "system", "radio", etc.
// AID_APP_START to AID_ISOLATED_START-1   -> u0_a1234
// AID_ISOLATED_START to AID_USER_OFFSET-1 -> u0_i1234
// AID_USER_OFFSET+                        -> u1_radio, u1_a1234, u2_i1234, etc.
// returns a passwd structure (sets errno to ENOENT on failure).
func appIDToPasswd(uid uint32) *Passwd {
	if uid < AidAppStart || !isValidAppID(uid, false) {
		// errno ENOENT
		return nil
	}

	var (
		dir   = "/data"
		name  = printAppNameFromUid(uid)
		appid = uid % AidUserOffset
		shell = "/bin/sh"
	)

	if appid < AidAppStart {
		dir = "/"
	}

	if _, err := os.Stat(shell); err != nil {
		shell = "/system/bin/sh"
	}

	var pw = Passwd{
		Name:  name,
		UID:   uid,
		GID:   uid,
		Dir:   dir,
		Shell: shell,
	}

	return &pw
}

// Translate a gid into the corresponding app_<gid>
// group structure (sets errno to ENOENT on failure).
func appIDToGroup(gid uint32) *Group {
	if gid < AidAppStart || !isValidAppID(gid, true) {
		// errno ENOENT
		return nil
	}

	var group = Group{
		Name: printAppNameFromGid(gid),
		GID:  gid,
	}

	return &group
}

func Getpwuid(uid uint32) *Passwd {
	if info := findAndroidIDInfoByID(uid); info != nil {
		return androidIInfoToPasswd(*info)
	}

	// Find an entry from the database file
	if pw := oemIDToPasswd(uid); pw != nil {
		return pw
	}

	return appIDToPasswd(uid)
}

func Getpwnam(login string) *Passwd {
	if info := findAndroidIDInfoByName(login); info != nil {
		return androidIInfoToPasswd(*info)
	}

	// Find an entry from the database file
	for _, file := range passwdFiles {
		// TODO implement
		_ = file
	}

	// Handle OEM range.
	if pw := oemIDToPasswd(oemIDFromName(login)); pw != nil {
		return pw
	}

	uid, err := appIDFromName(login, false)
	if err != nil {
		return nil
	}

	return appIDToPasswd(uid)
}

// GetGroupList All users are in just one group, the one passed in.
func GetGroupList() int {
	// TODO implement
	return 0
}

// GetLogin NOLINT: implementing bad function.
func GetLogin() string {
	// NOLINT: implementing bad function in terms of bad function.
	if pw := Getpwuid(uint32(os.Getuid())); pw != nil {
		return pw.Name
	}

	return ""
}

func Getgrgid(gid uint32) *Group {
	if info := findAndroidIDInfoByID(gid); info != nil {
		return androidIInfoToGroup(*info)
	}

	// Find an entry from the database file
	if group := oemIDToGroup(gid); group != nil {
		return group
	}

	return appIDToGroup(gid)
}

func Getgrnam(name string) *Group {
	if info := findAndroidIDInfoByName(name); info != nil {
		return androidIInfoToGroup(*info)
	}

	// Find an entry from the database file
	for _, file := range groupFiles {
		// TODO implement
		_ = file
	}

	// Handle OEM range.
	if group := oemIDToGroup(oemIDFromName(name)); group != nil {
		return group
	}

	gid, err := appIDFromName(name, true)
	if err != nil {
		return nil
	}

	return appIDToGroup(gid)
}
