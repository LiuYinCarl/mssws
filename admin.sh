# mssws admin script
# visit url: your_mssws_site/admin to run this script


admin_password="your_password"
blog_dir="./blog"
log_file="admin.log"

function update() {
    cd ${blog_dir}
    git pull
}

function usage() {
    echo "Usage:"
    echo "      url/admin/                              show admin help"
    echo "      url/admin/admin_password/update         update blog dir"
}

##########################################

# if log file do't exist, create it
if [ ! -f ${log_file} ];then
    touch ${log_file}
fi

# the visit time
now_date=`date`
# the visit info
visit_log=${now_date}" "$*
echo ${visit_log} >> ${log_file}


if [ ! $1 == ${admin_password} ]; then
    echo "wrong password"
    usage
    exit 0
fi

if [ $# -eq 2 ]; then
    if [ $2 == "update" ]; then
        update
    else
        usage
    fi
else
    usage
fi
