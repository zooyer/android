#!/bin/sh

echo 3 > /sys/class/remount/need_remount

mount -o remount,rw rootfs /
mount -o remount,rw /dev/block/system /system
mount -o remount,rw /dev/block/vendor /vendor