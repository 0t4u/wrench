# updater utility for me, pls ignore

curl -L -s -o ./wrench-update https://github.com/0t4u/wrench/releases/latest/download/wrench_linux_amd64; c=$?

if [[ $c -ne 0 ]]; then
    echo "failed to fetch wrench binary, wrench was not updated"
    rm ./wrench-update
else
    chmod +x ./wrench-update
    ./wrench-update version; ec=$?

    if [[ $ec -ne 0 ]]; then
        echo "something went wrong, wrench was not updated"
        rm ./wrench-update
    else
        mv ./wrench ./wrench.old
        mv ./wrench-update ./wrench
        echo "success, wrench was updated"
    fi
fi
