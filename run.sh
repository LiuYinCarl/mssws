if (( $EUID != 0 )); then
    echo "Please run as root to make sure mssws can bind 0.0.0.0:80 success."
    exit
fi

progress="mssws_prog"

echo "compile mssws_prog ..."
go build -o $progress main.go

pids=`ps -ef | grep ${progress} | grep -v grep | grep -v PPID | awk '{print $2}'`
for p in ${pids}
do
    echo "kill progress progress [ ${p} ]"
    kill ${p}
done

echo "start run mssws ..."
nohup ./${progress} &>/dev/null &

pids=`ps -ef | grep ${progress} | grep -v grep | grep -v PPID | awk '{print $2}'`
for p in ${pids}
do
    echo "run progress ${progress} [ ${p} ] success ..."
    exit
done
echo "run progress ${progress} failed ..."

