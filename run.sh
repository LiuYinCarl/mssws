#!/bin/bash

exec_name="mssws_prog"
go_src="main.go"
genindex_script="genindex.sh"

# save result of get_pids
g_pids=()

function f_get_pids() {
    g_pids=$(pgrep "${exec_name}")
    return 0
}

function f_check_root_rights() {
    if (( EUID != 0 )); then
	echo "Please run as root to make sure mssws can bind success."
	exit
    fi
}

function f_run_program() {
    # f_check_root_rights

    if [ ! -f ${exec_name} ]; then
	echo "${exec_name} don't exist, please run 'bash run.sh compile' first."
	exit
    fi

    f_get_pids
    for p in "${g_pids[@]}"; do
        if [[ "${p}" =~ ^[0-9]+$ ]]; then
	    echo "kill ${exec_name} [ ${p} ]."
	    kill "${p}"
        fi
    done

    echo "start run ${exec_name} ..."
    nohup ./${exec_name} &>server.log &

    f_get_pids
    for p in "${g_pids[@]}"; do
        if [[ "${p}" =~ ^[0-9]+$ ]]; then
	    echo "run ${exec_name} [ ${p} ] success ..."
            exit 0
        fi
    done
    echo "run ${exec_name} failed ..."
    exit -1
}


################################################################################

if [ ! -f ${genindex_script} ]; then
    echo "${genindex_script} not exist!"
    exit
else
    sh ${genindex_script}
    echo "generate index page data."
fi

if [ $# = 0 ]; then
    echo "compile ${exec_name} ..."
    go build -o ${exec_name} ${go_src}
    f_run_program
    exit
fi

if [ "$1" == "help" ]; then
    echo "run.sh Usage"
    echo "./run.sh help         show run.sh help"
    echo "./run.sh kill         kill ${exec_name}"
    echo "./run.sh compile      compile ${exec_name}"
    echo "./run.sh restart      kill and restart ${exec_name}"
    echo "./run.sh              compile ${exec_name} and restart ${exec_name}"

elif [ "$1" = "kill" ]; then
    f_get_pids
    for p in "${g_pids[@]}"; do
	echo "kill ${exec_name} [ ${p} ]"
	kill "${p}"
    done

elif [ "$1" = "compile" ]; then
    echo "compile ${exec_name} ..."
    go build -o ${exec_name} ${go_src}

elif [ "$1" = "restart" ]; then
    f_run_program

else
    echo "wrong argument. run 'bash run.sh help' to get more info"
fi

