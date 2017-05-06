#/bin/bash
set -e

if ! df -h | grep -Fq loop0; then
    echo "Need to make FS"
    losetup -d /dev/loop0
    dd if=/dev/zero of=filesystem.img bs=1 count=0 seek=8G
    sfdisk filesystem.img < filesystem-gen/layout.sfdisk
    losetup /dev/loop0 filesystem.img
    mkfs.fat /dev/loop0p1
    mkdir fat32 || echo "Unable to make fat32 dir, probs because it exists"
    mount /dev/loop0p1 ./fat32/
fi

cd fat32 && rm -r * || echo "wasnt able to remove things in the FS, probs because it was empty anyway"

# Zero out the drive so that it compresses better
for i in {1..8}
do
    dd if=/dev/zero of=$i.stuff bs=1 count=0 seek=1G || echo "."
done 
rm *.stuff
cd ..
./bin/filesystem-gen
umount ./fat32
losetup -d /dev/loop0
qemu-img convert -O qcow2 filesystem.img image.qcow2