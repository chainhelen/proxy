#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <sys/socket.h>

const int cpuNum = 20;
const int chNum = 2;

#define UNKNOW_PROCESS 0
#define MASTER_PROCESS 1
#define WORKER_PROCESS 2

struct Process {
    int channel[chNum];
    pid_t pid;
    int curProperty;
} process[cpuNum + 1];

void initProcess () {
    for(int i = 0;i < chNum;i++) {
        process[cpuNum].channel[i] = 0;
    }
    process[cpuNum].pid = getpid();
    process[cpuNum].curProperty = MASTER_PROCESS;
}

int main() {
    // fork
    pid_t pid;
    int index;
	initProcess();
    for (int i = 0;i < cpuNum;i++) {
        index = i;
        if(socketpair(AF_UNIX, SOCK_STREAM, 0, process[i].channel) < 0) {
            perror("socketpair failed");
            return -1;
        }
        // create child process
        pid = fork();
        if (0 == pid || -1 == pid) {
            break;
        }
    }
    if (-1 == pid) {
        perror("fork failed");
        exit(6);
    }
    if (0 == pid) {
        process[index].pid = getpid();
        process[index].curProperty = WORKER_PROCESS;
    } else {
        process[index].pid = pid;
        process[index].curProperty = WORKER_PROCESS;
    }
    return 0;
}
