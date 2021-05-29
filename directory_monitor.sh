#!/bin/bash

# source from https://www.cnblogs.com/kevingrace/p/8260032.html


# directory to save temp files
md5Dir='./.mssws_monitor'

# directory we need monitor
monitorDir=(
    ./blog/
)

# generate monitor directory's md5 if there is no md5old.log
OldFile () {
    for dir in ${monitorDir[@]}; do
	/bin/find ${dir} -type f | xargs md5sum >> ${md5Dir}/md5old.log
    done	
}

# generate monitor dirctory's md5 every we run the script
NewFile () {
    for dir in ${monitorDir[@]}; do
	/bin/find ${dir} -type f | xargs md5sum >> ${md5Dir}/md5new.log
    done
}


if [ ! -d ${md5Dir} ]; then
    mkdir ${md5Dir}
fi 

# if there is no md5old.log file, generate it
if [ ! -f ${md5Dir}/md5old.log ]; then
    OldFile
fi

# generate md5new.log
NewFile

/usr/bin/diff ${md5Dir}/md5new.log ${md5Dir}/md5old.log > ${md5Dir}/md5diff.log
Status=$?

if [ ${Status} -ne 0 ]; then
    bash genindex.sh
fi

cat ${md5Dir}/md5new.log > ${md5Dir}/md5old.log
cat /dev/null > ${md5Dir}/md5new.log
